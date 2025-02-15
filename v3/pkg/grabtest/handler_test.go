package grabtest

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestHandlerDefaults(t *testing.T) {
	WithTestServer(t, func(url string) {
		resp := MustHTTPDo(MustHTTPNewRequest("GET", url, nil))
		AssertHTTPResponseStatusCode(t, resp, http.StatusOK)
		AssertHTTPResponseContentLength(t, resp, 1048576)
		AssertHTTPResponseHeader(t, resp, "Accept-Ranges", "bytes")
	})
}

func TestHandlerMethodWhitelist(t *testing.T) {
	tests := []struct {
		Whitelist        []string
		Method           string
		ExpectStatusCode int
	}{
		{Whitelist: []string{"GET", "HEAD"}, Method: "GET", ExpectStatusCode: http.StatusOK},
		{Whitelist: []string{"GET", "HEAD"}, Method: "HEAD", ExpectStatusCode: http.StatusOK},
		{Whitelist: []string{"GET"}, Method: "HEAD", ExpectStatusCode: http.StatusMethodNotAllowed},
		{Whitelist: []string{"HEAD"}, Method: "GET", ExpectStatusCode: http.StatusMethodNotAllowed},
	}

	for _, test := range tests {
		WithTestServer(t, func(url string) {
			resp := MustHTTPDoWithClose(MustHTTPNewRequest(test.Method, url, nil))
			AssertHTTPResponseStatusCode(t, resp, test.ExpectStatusCode)
		}, MethodWhitelist(test.Whitelist...))
	}
}

func TestHandlerHeaderBlacklist(t *testing.T) {
	contentLength := 4096
	WithTestServer(t, func(url string) {
		resp := MustHTTPDo(MustHTTPNewRequest("GET", url, nil))
		defer resp.Body.Close()
		if resp.ContentLength != -1 {
			t.Errorf("expected Response.ContentLength: -1, got: %d", resp.ContentLength)
		}
		AssertHTTPResponseHeader(t, resp, "Content-Length", "")
		AssertHTTPResponseBodyLength(t, resp, int64(contentLength))
	},
		ContentLength(contentLength),
		HeaderBlacklist("Content-Length"),
	)
}

func TestHandlerStatusCodeFuncs(t *testing.T) {
	expect := 418 // I'm a teapot
	WithTestServer(t, func(url string) {
		resp := MustHTTPDo(MustHTTPNewRequest("GET", url, nil))
		defer resp.Body.Close()
		AssertHTTPResponseStatusCode(t, resp, expect)
	},
		StatusCode(func(req *http.Request) int { return expect }),
	)
}

func TestHandlerContentLength(t *testing.T) {
	tests := []struct {
		Method          string
		ContentLength   int
		ExpectHeaderLen int64
		ExpectBodyLen   int
	}{
		{Method: "GET", ContentLength: 321, ExpectHeaderLen: 321, ExpectBodyLen: 321},
		{Method: "HEAD", ContentLength: 321, ExpectHeaderLen: 321, ExpectBodyLen: 0},
		{Method: "GET", ContentLength: 0, ExpectHeaderLen: 0, ExpectBodyLen: 0},
		{Method: "HEAD", ContentLength: 0, ExpectHeaderLen: 0, ExpectBodyLen: 0},
	}

	for _, test := range tests {
		WithTestServer(t, func(url string) {
			resp := MustHTTPDo(MustHTTPNewRequest(test.Method, url, nil))
			defer resp.Body.Close()

			AssertHTTPResponseHeader(t, resp, "Content-Length", "%d", test.ExpectHeaderLen)

			b, err := io.ReadAll(resp.Body)
			if err != nil {
				panic(err)
			}
			if len(b) != test.ExpectBodyLen {
				t.Errorf(
					"expected body length: %v, got: %v, in: %v",
					test.ExpectBodyLen,
					len(b),
					test,
				)
			}
		},
			ContentLength(test.ContentLength),
		)
	}
}

func TestHandlerAcceptRanges(t *testing.T) {
	header := "Accept-Ranges"
	n := 128
	t.Run("Enabled", func(t *testing.T) {
		WithTestServer(t, func(url string) {
			req := MustHTTPNewRequest("GET", url, nil)
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", n/2))
			resp := MustHTTPDo(req)
			AssertHTTPResponseHeader(t, resp, header, "bytes")
			AssertHTTPResponseContentLength(t, resp, int64(n/2))
		},
			ContentLength(n),
		)
	})

	t.Run("Disabled", func(t *testing.T) {
		WithTestServer(t, func(url string) {
			req := MustHTTPNewRequest("GET", url, nil)
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", n/2))
			resp := MustHTTPDo(req)
			AssertHTTPResponseHeader(t, resp, header, "")
			AssertHTTPResponseContentLength(t, resp, int64(n))
		},
			AcceptRanges(false),
			ContentLength(n),
		)
	})
}

func TestHandlerAttachmentFilename(t *testing.T) {
	filename := "foo.pdf"
	WithTestServer(t, func(url string) {
		resp := MustHTTPDoWithClose(MustHTTPNewRequest("GET", url, nil))
		AssertHTTPResponseHeader(t, resp, "Content-Disposition", `attachment;filename="%s"`, filename)
	},
		AttachmentFilename(filename),
	)
}

func TestHandlerLastModified(t *testing.T) {
	WithTestServer(t, func(url string) {
		resp := MustHTTPDoWithClose(MustHTTPNewRequest("GET", url, nil))
		AssertHTTPResponseHeader(t, resp, "Last-Modified", "Thu, 29 Nov 1973 21:33:09 GMT")
	},
		LastModified(time.Unix(123456789, 0)),
	)
}
