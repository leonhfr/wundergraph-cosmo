package compression

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/go-chi/chi/v5"
)

func TestMiddleware(t *testing.T) {
	graphqlPath := "/graphql"
	testStr := "{\"data\":{\"foo\":\"bar\"}}"

	r := chi.NewRouter()
	r.Use(NewMiddleware())
	r.Post(graphqlPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.Write([]byte(testStr))
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	tests := []struct {
		name              string
		acceptedEncodings []string
		expectedEncoding  string
	}{
		{
			name:              "no expected encodings due to no accepted encodings",
			acceptedEncodings: nil,
			expectedEncoding:  "",
		},
		{
			name:              "brotli is used",
			acceptedEncodings: []string{"br"},
			expectedEncoding:  "br",
		},
		{
			name:              "gzip is used",
			acceptedEncodings: []string{"gzip"},
			expectedEncoding:  "gzip",
		},
		{
			name:              "deflate is used",
			acceptedEncodings: []string{"deflate"},
			expectedEncoding:  "deflate",
		},
		{
			name:              "gzip is preferred over deflate",
			acceptedEncodings: []string{"gzip", "deflate"},
			expectedEncoding:  "gzip",
		},
		{
			name:              "brotli is preferred over gzip",
			acceptedEncodings: []string{"br", "gzip"},
			expectedEncoding:  "br",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, respString := testRequestWithAcceptedEncodings(t, ts, "POST", graphqlPath, tt.acceptedEncodings...)
			if respString != testStr {
				t.Errorf("expected response %q but got %q", testStr, respString)
			}
			if got := resp.Header.Get("Content-Encoding"); got != tt.expectedEncoding {
				t.Errorf("expected encoding %q but got %q", tt.expectedEncoding, got)
			}
		})
	}
}

func testRequestWithAcceptedEncodings(t *testing.T, ts *httptest.Server, method, path string, encodings ...string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, nil)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}
	if len(encodings) > 0 {
		encodingsString := strings.Join(encodings, ",")
		req.Header.Set("Accept-Encoding", encodingsString)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}

	respBody := decodeResponseBody(t, resp)
	defer resp.Body.Close()

	return resp, respBody
}

func decodeResponseBody(t *testing.T, resp *http.Response) string {
	switch resp.Header.Get("Content-Encoding") {
	case "br":
		reader := brotli.NewReader(resp.Body)
		return readAll(t, reader)
	case "gzip":
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		return readAll(t, reader)
	case "deflate":
		reader := flate.NewReader(resp.Body)
		defer reader.Close()
		return readAll(t, reader)
	default:
		return readAll(t, resp.Body)
	}
}

func readAll(t *testing.T, r io.Reader) string {
	bytes, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
		return ""
	}
	return string(bytes)
}

// 	tests := []struct {
// 		name              string
// 		path              string
// 		expectedEncoding  string
// 		acceptedEncodings []string
// 	}{
// 		{
// 			name:              "no expected encodings due to no accepted encodings",
// 			path:              "/gethtml",
// 			acceptedEncodings: nil,
// 			expectedEncoding:  "",
// 		},
// 		{
// 			name:              "no expected encodings due to content type",
// 			path:              "/getplain",
// 			acceptedEncodings: nil,
// 			expectedEncoding:  "",
// 		},
// 		{
// 			name:              "gzip is only encoding",
// 			path:              "/gethtml",
// 			acceptedEncodings: []string{"gzip"},
// 			expectedEncoding:  "gzip",
// 		},
// 		{
// 			name:              "gzip is preferred over deflate",
// 			path:              "/getcss",
// 			acceptedEncodings: []string{"gzip", "deflate"},
// 			expectedEncoding:  "gzip",
// 		},
// 		{
// 			name:              "deflate is used",
// 			path:              "/getcss",
// 			acceptedEncodings: []string{"deflate"},
// 			expectedEncoding:  "deflate",
// 		},
// 		{

// 			name:              "nop is preferred",
// 			path:              "/getcss",
// 			acceptedEncodings: []string{"nop, gzip, deflate"},
// 			expectedEncoding:  "nop",
// 		},
// 	}

// 	for _, tc := range tests {
// 		tc := tc
// 		t.Run(tc.name, func(t *testing.T) {
// 			resp, respString := testRequestWithAcceptedEncodings(t, ts, "GET", tc.path, tc.acceptedEncodings...)
// 			if respString != "textstring" {
// 				t.Errorf("response text doesn't match; expected:%q, got:%q", "textstring", respString)
// 			}
// 			if got := resp.Header.Get("Content-Encoding"); got != tc.expectedEncoding {
// 				t.Errorf("expected encoding %q but got %q", tc.expectedEncoding, got)
// 			}

// 		})

// 	}
// }

// func TestCompressorWildcards(t *testing.T) {
// 	tests := []struct {
// 		name       string
// 		recover    string
// 		types      []string
// 		typesCount int
// 		wcCount    int
// 	}{
// 		{
// 			name:       "defaults",
// 			typesCount: 10,
// 		},
// 		{
// 			name:       "no wildcard",
// 			types:      []string{"text/plain", "text/html"},
// 			typesCount: 2,
// 		},
// 		{
// 			name:    "invalid wildcard #1",
// 			types:   []string{"audio/*wav"},
// 			recover: "middleware/compress: Unsupported content-type wildcard pattern 'audio/*wav'. Only '/*' supported",
// 		},
// 		{
// 			name:    "invalid wildcard #2",
// 			types:   []string{"application*/*"},
// 			recover: "middleware/compress: Unsupported content-type wildcard pattern 'application*/*'. Only '/*' supported",
// 		},
// 		{
// 			name:    "valid wildcard",
// 			types:   []string{"text/*"},
// 			wcCount: 1,
// 		},
// 		{
// 			name:       "mixed",
// 			types:      []string{"audio/wav", "text/*"},
// 			typesCount: 1,
// 			wcCount:    1,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			defer func() {
// 				if tt.recover == "" {
// 					tt.recover = "<nil>"
// 				}
// 				if r := recover(); tt.recover != fmt.Sprintf("%v", r) {
// 					t.Errorf("Unexpected value recovered: %v", r)
// 				}
// 			}()
// 			compressor := NewCompressor(5, tt.types...)
// 			if len(compressor.allowedTypes) != tt.typesCount {
// 				t.Errorf("expected %d allowedTypes, got %d", tt.typesCount, len(compressor.allowedTypes))
// 			}
// 			if len(compressor.allowedWildcards) != tt.wcCount {
// 				t.Errorf("expected %d allowedWildcards, got %d", tt.wcCount, len(compressor.allowedWildcards))
// 			}
// 		})
// 	}
// }
