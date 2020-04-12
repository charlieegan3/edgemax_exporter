package edgemax

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

func wsMarshal(v interface{}) ([]byte, byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, 0, err
	}

	blen := []byte(strconv.Itoa(len(b)) + "\n")
	return append(blen, b...), 0, nil
}

func wsUnmarshal(data []byte, _ byte, v interface{}) error {
	if data[0] == '{' {
		return json.Unmarshal(data, v)
	}

	bb := bytes.SplitN(data, []byte("\n"), 2)
	if l := len(bb); l != 2 {
		return fmt.Errorf("incorrect number of elements in websocket message: %d", l)
	}

	if len(bb[1]) == 0 {
		return nil
	}

	return json.Unmarshal(bb[1], v)
}

type wsName struct {
	Name StatType `json:"name"`
}

type wsRequest struct {
	Subscribe   []wsName `json:"SUBSCRIBE"`
	Unsubscribe []wsName `json:"UNSUBSCRIBE"`
	SessionID   string   `json:"SESSION_ID"`
}
