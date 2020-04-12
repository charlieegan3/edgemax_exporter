package edgemax

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
)

func Test_wsMarshal(t *testing.T) {
	var tests = []struct {
		desc string
		wsr  wsRequest
		out  []byte
	}{
		{
			desc: "empty request",
			wsr:  wsRequest{},
			out:  append([]byte("53\n"), `{"SUBSCRIBE":null,"UNSUBSCRIBE":null,"SESSION_ID":""}`...),
		},
		{
			desc: "subscribe to two streams",
			wsr: wsRequest{
				Subscribe: []wsName{
					{Name: "foo"},
					{Name: "bar"},
				},
				SessionID: "baz",
			},
			out: append([]byte("83\n"), `{"SUBSCRIBE":[{"name":"foo"},{"name":"bar"}],"UNSUBSCRIBE":null,"SESSION_ID":"baz"}`...),
		},
		{
			desc: "unsubscribe from one stream",
			wsr: wsRequest{
				Unsubscribe: []wsName{
					{Name: "foo"},
				},
				SessionID: "bar",
			},
			out: append([]byte("68\n"), `{"SUBSCRIBE":null,"UNSUBSCRIBE":[{"name":"foo"}],"SESSION_ID":"bar"}`...),
		},
	}

	for i, tt := range tests {
		t.Logf("[%02d] test %q", i, tt.desc)

		out, pType, err := wsMarshal(tt.wsr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if want, got := byte(0), pType; want != got {
			t.Fatalf("unexpected payload type:\n- want: %v\n-  got: %v", want, got)
		}

		if want, got := tt.out, out; !bytes.Equal(want, got) {
			t.Fatalf("unexpected output:\n- want: %q\n-  got: %q", string(want), string(got))
		}
	}
}

func Test_wsUnmarshal(t *testing.T) {
	var tests = []struct {
		desc string
		in   []byte
		wsr  wsRequest
		err  error
	}{
		{
			desc: "incorrect number of newlines",
			in:   []byte("foo"),
			err:  errors.New("incorrect number of elements in websocket message: 1"),
		},
		{
			desc: "no JSON object present",
			in:   []byte("3\n"),
			wsr:  wsRequest{},
		},
		{
			desc: "empty request with no length",
			in:   []byte(`{"SUBSCRIBE":null,"UNSUBSCRIBE":null,"SESSION_ID":""}`),
			wsr:  wsRequest{},
		},
		{
			desc: "empty request",
			in:   append([]byte("53\n"), `{"SUBSCRIBE":null,"UNSUBSCRIBE":null,"SESSION_ID":""}`...),
			wsr:  wsRequest{},
		},
		{
			desc: "subscribe to two streams",
			in:   append([]byte("83\n"), `{"SUBSCRIBE":[{"name":"foo"},{"name":"bar"}],"UNSUBSCRIBE":null,"SESSION_ID":"baz"}`...),
			wsr: wsRequest{
				Subscribe: []wsName{
					{Name: "foo"},
					{Name: "bar"},
				},
				SessionID: "baz",
			},
		},
		{
			desc: "unsubscribe from one stream",
			in:   append([]byte("68\n"), `{"SUBSCRIBE":null,"UNSUBSCRIBE":[{"name":"foo"}],"SESSION_ID":"bar"}`...),
			wsr: wsRequest{
				Unsubscribe: []wsName{
					{Name: "foo"},
				},
				SessionID: "bar",
			},
		},
	}

	for i, tt := range tests {
		t.Logf("[%02d] test %q", i, tt.desc)

		var wsr wsRequest
		err := wsUnmarshal(tt.in, 0, &wsr)
		if want, got := errStr(tt.err), errStr(err); want != got {
			t.Fatalf("unexpected error:\n- want: %v\n-  got: %v", want, got)
		}
		if err != nil {
			continue
		}

		if want, got := tt.wsr, wsr; !reflect.DeepEqual(want, got) {
			t.Fatalf("unexpected wsRequest object:\n- want: %v\n-  got: %v", want, got)
		}
	}
}

func errStr(err error) string {
	if err == nil {
		return "<nil>"
	}

	return err.Error()
}
