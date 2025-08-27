// nolint:exhaustruct
package zen

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSession_BindQuery(t *testing.T) {
	tests := []struct {
		name        string
		queryString string
		target      interface{}
		expected    interface{}
		wantErr     bool
		errSubstr   string
	}{
		{
			name:        "basic types",
			queryString: "str=hello&num=42&flag=true&float=3.14",
			target: &struct {
				Str   string  `json:"str"`
				Num   int     `json:"num"`
				Flag  bool    `json:"flag"`
				Float float64 `json:"float"`
			}{},
			expected: &struct {
				Str   string  `json:"str"`
				Num   int     `json:"num"`
				Flag  bool    `json:"flag"`
				Float float64 `json:"float"`
			}{
				Str:   "hello",
				Num:   42,
				Flag:  true,
				Float: 3.14,
			},
			wantErr: false,
		},
		{
			name:        "array params",
			queryString: "tags=one&tags=two&tags=three&nums=1&nums=2&nums=3",
			target: &struct {
				Tags []string `json:"tags"`
				Nums []int    `json:"nums"`
			}{},
			expected: &struct {
				Tags []string `json:"tags"`
				Nums []int    `json:"nums"`
			}{
				Tags: []string{"one", "two", "three"},
				Nums: []int{1, 2, 3},
			},
			wantErr: false,
		},
		{
			name:        "missing params",
			queryString: "name=test",
			target: &struct {
				Name  string `json:"name"`
				Count int    `json:"count"`
			}{},
			expected: &struct {
				Name  string `json:"name"`
				Count int    `json:"count"`
			}{
				Name:  "test",
				Count: 0, // Default value retained
			},
			wantErr: false,
		},
		{
			name:        "ignored fields",
			queryString: "name=test&secret=hidden&internal=value",
			target: &struct {
				Name     string `json:"name"`
				Secret   string `json:"-"`
				Internal string // No json tag
			}{},
			expected: &struct {
				Name     string `json:"name"`
				Secret   string `json:"-"`
				Internal string
			}{
				Name:     "test",
				Secret:   "", // Not set from query
				Internal: "", // Not set from query
			},
			wantErr: false,
		},
		{
			name:        "type error",
			queryString: "num=not-a-number",
			target: &struct {
				Num int `json:"num"`
			}{},
			wantErr:   true,
			errSubstr: "could not parse num as integer",
		},
		{
			name:        "boolean error",
			queryString: "flag=not-a-boolean",
			target: &struct {
				Flag bool `json:"flag"`
			}{},
			wantErr:   true,
			errSubstr: "could not parse flag as boolean",
		},
		{
			name:        "nil target",
			queryString: "name=test",
			target:      nil,
			wantErr:     true,
			errSubstr:   "destination must be a non-nil pointer",
		},
		{
			name:        "non-pointer target",
			queryString: "name=test",
			target: struct {
				Name string `json:"name"`
			}{},
			wantErr:   true,
			errSubstr: "destination must be a non-nil pointer",
		},
		{
			name:        "pointer to non-struct",
			queryString: "value=test",
			target:      new(string),
			wantErr:     true,
			errSubstr:   "destination must be a pointer to a struct",
		},
		{
			name:        "uint type",
			queryString: "count=123",
			target: &struct {
				Count uint `json:"count"`
			}{},
			expected: &struct {
				Count uint `json:"count"`
			}{
				Count: 123,
			},
			wantErr: false,
		},
		{
			name:        "mixed slice types",
			queryString: "nums=1&nums=2&nums=3&flags=true&flags=false",
			target: &struct {
				Nums  []int  `json:"nums"`
				Flags []bool `json:"flags"`
			}{},
			expected: &struct {
				Nums  []int  `json:"nums"`
				Flags []bool `json:"flags"`
			}{
				Nums:  []int{1, 2, 3},
				Flags: []bool{true, false},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request with the query string
			req := httptest.NewRequest(http.MethodGet, "/?"+tt.queryString, nil)

			// Create a session
			sess := &Session{
				r: req,
			}

			// Call BindQuery
			err := sess.BindQuery(tt.target)

			// Check error conditions
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
				return
			}

			// Verify no error for positive cases
			require.NoError(t, err)

			// Check that the struct was populated correctly
			assert.Equal(t, tt.expected, tt.target)
		})
	}
}

func TestSession_BindQuery_EdgeCases(t *testing.T) {
	t.Run("empty query string", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		sess := &Session{r: req}

		target := &struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}{}

		err := sess.BindQuery(target)
		require.NoError(t, err)
		assert.Equal(t, "", target.Name)
		assert.Equal(t, 0, target.Age)
	})

	t.Run("url encoded values", func(t *testing.T) {
		queryValues := url.Values{}
		queryValues.Add("name", "John Doe")
		queryValues.Add("tag", "special character: &")

		req := httptest.NewRequest(http.MethodGet, "/?"+queryValues.Encode(), nil)
		sess := &Session{r: req}

		target := &struct {
			Name string `json:"name"`
			Tag  string `json:"tag"`
		}{}

		err := sess.BindQuery(target)
		require.NoError(t, err)
		assert.Equal(t, "John Doe", target.Name)
		assert.Equal(t, "special character: &", target.Tag)
	})

	t.Run("field with json tag options", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?name=test&count=10", nil)
		sess := &Session{r: req}

		target := &struct {
			Name  string `json:"name,omitempty"`
			Count int    `json:"count,string"`
		}{}

		err := sess.BindQuery(target)
		require.NoError(t, err)
		assert.Equal(t, "test", target.Name)
		assert.Equal(t, 10, target.Count)
	})
}

func TestSession_BindQuery_Init(t *testing.T) {
	t.Run("verify session initialization and reuse", func(t *testing.T) {
		// Create a request with query params
		req := httptest.NewRequest(http.MethodGet, "/?name=test&age=30", nil)
		w := httptest.NewRecorder()

		// Create and initialize a session
		sess := &Session{}
		err := sess.init(w, req, 0)
		require.NoError(t, err)

		// Bind query params
		target := &struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}{}

		err = sess.BindQuery(target)
		require.NoError(t, err)
		assert.Equal(t, "test", target.Name)
		assert.Equal(t, 30, target.Age)

		// Reset and verify it's clean
		sess.reset()
		assert.Nil(t, sess.r)
	})
}
