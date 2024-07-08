package validate

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestURLSearchParamsInto(t *testing.T) {
	v := Validate{Instance: validator.New()}

	tests := []struct {
		name       string
		url        string
		dest       func() interface{}
		check      func(interface{}) bool
		shouldFail bool
	}{
		{
			name: "Basic types",
			url:  "http://example.com?name=John&age=30&active=true&height=1.75",
			dest: func() interface{} {
				return &struct {
					Name   string  `json:"name"`
					Age    int     `json:"age"`
					Active bool    `json:"active"`
					Height float64 `json:"height"`
				}{}
			},
			check: func(i interface{}) bool {
				d := i.(*struct {
					Name   string  `json:"name"`
					Age    int     `json:"age"`
					Active bool    `json:"active"`
					Height float64 `json:"height"`
				})
				return d.Name == "John" && d.Age == 30 && d.Active == true && d.Height == 1.75
			},
		},
		{
			name: "Slice of strings",
			url:  "http://example.com?tags=go&tags=programming&tags=test",
			dest: func() interface{} {
				return &struct {
					Tags []string `json:"tags"`
				}{}
			},
			check: func(i interface{}) bool {
				d := i.(*struct {
					Tags []string `json:"tags"`
				})
				return reflect.DeepEqual(d.Tags, []string{"go", "programming", "test"})
			},
		},
		{
			name: "Mixed types",
			url:  "http://example.com?name=Alice&age=25&scores=90&scores=85&scores=95",
			dest: func() interface{} {
				return &struct {
					Name   string `json:"name"`
					Age    int    `json:"age"`
					Scores []int  `json:"scores"`
				}{}
			},
			check: func(i interface{}) bool {
				d := i.(*struct {
					Name   string `json:"name"`
					Age    int    `json:"age"`
					Scores []int  `json:"scores"`
				})
				return d.Name == "Alice" && d.Age == 25 && reflect.DeepEqual(d.Scores, []int{90, 85, 95})
			},
		},
		{
			name: "Validation failure",
			url:  "http://example.com?email=invalid-email",
			dest: func() interface{} {
				return &struct {
					Email string `json:"email" validate:"email"`
				}{}
			},
			shouldFail: true,
		},
		{
			name: "Missing required field",
			url:  "http://example.com",
			dest: func() interface{} {
				return &struct {
					Name string `json:"name" validate:"required"`
				}{}
			},
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := http.NewRequest("GET", tt.url, nil)
			dest := tt.dest()
			err := v.URLSearchParamsInto(r, dest)

			if tt.shouldFail {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if tt.check != nil && !tt.check(dest) {
					t.Errorf("Check failed for %+v", dest)
				}
			}
		})
	}
}
