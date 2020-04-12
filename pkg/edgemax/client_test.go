package edgemax

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestClientRetainsCookies(t *testing.T) {
	const cookieName = "foo"
	wantCookie := &http.Cookie{
		Name:  cookieName,
		Value: "bar",
	}

	var i int
	c, done := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		defer func() { i++ }()

		switch i {
		case 0:
			http.SetCookie(w, wantCookie)
		case 1:
			c, err := r.Cookie(cookieName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := wantCookie, c; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected cookie:\n- want: %v\n-  got: %v",
					want, got)
			}
		}

		_, _ = w.Write([]byte(`{}`))
	})
	defer done()

	for i := 0; i < 2; i++ {
		req, err := c.newRequest(http.MethodGet, "/")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = c.do(req, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestClientLogin(t *testing.T) {
	const (
		wantUsername = "username"
		wantPassword = "password"
	)

	h := testHandler(t, http.MethodPost, "/")
	c, done := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		h(w, r)

		username := r.PostFormValue("username")
		password := r.PostFormValue("password")

		if want, got := wantUsername, username; want != got {
			t.Fatalf("unexpected username:\n- want: %v\n-  got: %v", want, got)
		}

		if want, got := wantPassword, password; want != got {
			t.Fatalf("unexpected password:\n- want: %v\n-  got: %v", want, got)
		}
	})
	defer done()

	if err := c.Login(wantUsername, wantPassword); err != nil {
		t.Fatalf("unexpected error from Client.Login: %v", err)
	}
}

func TestInsecureHTTPClient(t *testing.T) {
	timeout := 5 * time.Second
	c := InsecureHTTPClient(timeout)

	if want, got := c.Timeout, timeout; want != got {
		t.Fatalf("unexpected client timeout:\n- want: %v\n-  got: %v",
			want, got)
	}

	got := c.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify
	if want := true; want != got {
		t.Fatalf("unexpected client insecure skip verify value:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func testClient(t *testing.T, fn func(w http.ResponseWriter, r *http.Request)) (*Client, func()) {
	s := httptest.NewServer(http.HandlerFunc(fn))

	c, err := NewClient(s.URL, nil)
	if err != nil {
		t.Fatalf("error creating Client: %v", err)
	}

	return c, func() { s.Close() }
}

func testHandler(t *testing.T, method string, path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if want, got := method, r.Method; want != got {
			t.Fatalf("unexpected HTTP method:\n- want: %v\n-  got: %v", want, got)
		}

		if want, got := path, r.URL.Path; want != got {
			t.Fatalf("unexpected URL path:\n- want: %v\n-  got: %v", want, got)
		}
	}
}
