package edgemax

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

// Stats opens a websocket connection to an EdgeMAX device to retrieve
// statistics which are sent using the socket.  The done closure must
// be invoked to clean up resources from Stats.
func (c *Client) Stats(stats ...StatType) (statC chan Stat, done func() error, err error) {
	if stats == nil {
		stats = []StatType{
			StatTypeDPIStats,
			StatTypeInterfaces,
			StatTypeSystemStats,
		}
	}

	doneC := make(chan struct{})
	errC := make(chan error)
	wg := new(sync.WaitGroup)

	wg.Add(1)
	go func() {
		defer func() {
			close(errC)
			wg.Done()
		}()

		if err := c.keepalive(doneC); err != nil {
			errC <- err
		}
	}()

	statC, wsDone, err := c.initWebsocket(stats)
	if err != nil {
		return nil, nil, err
	}

	done = func() error {
		close(doneC)
		wg.Wait()

		if err := <-errC; err != nil {
			return err
		}

		if err := wsDone(); err != nil {
			return err
		}

		close(statC)
		return nil
	}

	return statC, done, nil
}

const (
	// sessionCookie is the name of the session cookie used to authenticate
	// against EdgeMAX devices.
	sessionCookie = "PHPSESSID"
)

// initWebsocket initializes the websocket used for Client.Stats, and provides
// a closure which can be used to clean it up.
func (c *Client) initWebsocket(stats []StatType) (chan Stat, func() error, error) {
	// Websocket URL is adapted from HTTP URL
	wsURL := *c.apiURL
	wsURL.Scheme = "wss"
	wsURL.Path = "/ws/stats"

	cfg, err := websocket.NewConfig(wsURL.String(), c.apiURL.String())
	if err != nil {
		return nil, nil, err
	}

	// Copy TLS config from client if using standard *http.Transport, so that
	// using InsecureHTTPClient can also apply to websocket connections
	if tr, ok := c.client.Transport.(*http.Transport); ok {
		cfg.TlsConfig = tr.TLSClientConfig
	}

	// Need session ID from cookie to pass as part of websocket subscription
	var sessionID string
	for _, c := range c.client.Jar.Cookies(c.apiURL) {
		if c.Name == sessionCookie {
			sessionID = c.Value
			break
		}
	}

	wsc, err := websocket.DialConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	wsCodec := &websocket.Codec{
		Marshal:   wsMarshal,
		Unmarshal: wsUnmarshal,
	}

	wsns := make([]wsName, 0, len(stats))
	for _, stat := range stats {
		wsns = append(wsns, wsName{Name: stat})
	}

	// Subscribe to stats named in StatType slice, authenticate using session
	// cookie value
	sub := &wsRequest{
		Subscribe: wsns,
		SessionID: sessionID,
	}

	if err := wsCodec.Send(wsc, sub); err != nil {
		return nil, nil, err
	}

	statC := make(chan Stat)
	doneC := make(chan struct{})

	wg := new(sync.WaitGroup)

	// Unsubscribe and clean up websocket on completion using clsosure
	done := statsDone(wg, sub, wsCodec, wsc, doneC)

	// Collect raw stats from websocket, parse them, and send them into statC
	wg.Add(1)
	go collectStats(wg, wsCodec, wsc, statC, doneC)

	return statC, done, nil
}

// keepalive sends heartbeat requests at regular intervals to the EdgeMAX
// device to keep a session active while Client.Stats is running.
func (c *Client) keepalive(doneC <-chan struct{}) error {
	var v struct {
		Success bool `json:"success"`
		Ping    bool `json:"PING"`
		Session bool `json:"SESSION"`
	}

	for {
		req, err := c.newRequest(
			http.MethodGet,
			fmt.Sprintf("/api/edge/heartbeat.json?_=%d", time.Now().UnixNano()),
		)
		if err != nil {
			return err
		}

		_, err = c.do(req, &v)
		if err != nil {
			return err
		}

		select {
		case <-time.After(5 * time.Second):
		case <-doneC:
			return nil
		}
	}
}

// statsDone creates a closure which can be invoked to unsubscribe from the
// stats websocket and close the connection gracefully.
func statsDone(
	wg *sync.WaitGroup,
	sub *wsRequest,
	wsCodec *websocket.Codec,
	wsc *websocket.Conn,
	doneC chan<- struct{},
) func() error {
	return func() error {
		// Unsubscribe from the same stats that were subscribed to
		names := make([]wsName, len(sub.Subscribe))
		copy(names, sub.Subscribe)
		sub.Unsubscribe = names
		sub.Subscribe = nil

		if err := wsCodec.Send(wsc, sub); err != nil {
			return err
		}

		if err := wsc.Close(); err != nil {
			return err
		}

		// Halt stats collection goroutine
		close(doneC)
		wg.Wait()

		return nil
	}
}

// collectStats receives raw stats from a websocket and decodes them into
// Stat structs of various types.
func collectStats(
	wg *sync.WaitGroup,
	wsCodec *websocket.Codec,
	wsc *websocket.Conn,
	statC chan<- Stat,
	doneC chan struct{},
) {
	for {
		select {
		case <-doneC:
			wg.Done()
			return
		default:
		}

		m := make(map[StatType]json.RawMessage)
		if err := wsCodec.Receive(wsc, &m); err != nil {
			continue
		}

		for k, v := range m {
			switch k {
			case StatTypeDPIStats:
				var ds DPIStats
				if err := ds.UnmarshalJSON(v); err != nil {
					break
				}

				statC <- ds
			case StatTypeInterfaces:
				var is Interfaces
				if err := is.UnmarshalJSON(v); err != nil {
					break
				}

				statC <- is
			case StatTypeSystemStats:
				ss := new(SystemStats)
				if err := ss.UnmarshalJSON(v); err != nil {
					break
				}

				statC <- ss
			}
		}
	}
}
