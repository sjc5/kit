package validate

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

type Embedded struct {
	EmbeddedField string `json:"embeddedField"`
}

type DoubleEmbedded struct {
	Embedded
	EmbeddedField2 string `json:"embeddedField2"`
}

func TestURLSearchParamsInto(t *testing.T) {
	v := New()

	tests := []struct {
		name       string
		url        string
		dest       func() any
		check      func(any) bool
		shouldFail bool
	}{
		{
			name: "Basic types",
			url:  "http://example.com?name=John&age=30&active=true&height=1.75",
			dest: func() any {
				return &struct {
					Name   string  `json:"name"`
					Age    int     `json:"age"`
					Active bool    `json:"active"`
					Height float64 `json:"height"`
				}{}
			},
			check: func(i any) bool {
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
			dest: func() any {
				return &struct {
					Tags []string `json:"tags"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					Tags []string `json:"tags"`
				})
				return reflect.DeepEqual(d.Tags, []string{"go", "programming", "test"})
			},
		},
		{
			name: "Mixed types",
			url:  "http://example.com?name=Alice&age=25&scores=90&scores=85&scores=95",
			dest: func() any {
				return &struct {
					Name   string `json:"name"`
					Age    int    `json:"age"`
					Scores []int  `json:"scores"`
				}{}
			},
			check: func(i any) bool {
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
			dest: func() any {
				return &struct {
					Email string `json:"email" validate:"email"`
				}{}
			},
			shouldFail: true,
		},
		{
			name: "Missing required field",
			url:  "http://example.com",
			dest: func() any {
				return &struct {
					Name string `json:"name" validate:"required"`
				}{}
			},
			shouldFail: true,
		},
		{
			name: "Pointer fields",
			url:  "http://example.com?name=Jane&age=28&salary=50000.50&isEmployee=true",
			dest: func() any {
				return &struct {
					Name       *string  `json:"name"`
					Age        *int     `json:"age"`
					Salary     *float64 `json:"salary"`
					IsEmployee *bool    `json:"isEmployee"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					Name       *string  `json:"name"`
					Age        *int     `json:"age"`
					Salary     *float64 `json:"salary"`
					IsEmployee *bool    `json:"isEmployee"`
				})

				if d.Name == nil || *d.Name != "Jane" {
					fmt.Printf("Name: expected 'Jane', got %v\n", d.Name)
					return false
				}
				if d.Age == nil || *d.Age != 28 {
					fmt.Printf("Age: expected 28, got %v\n", d.Age)
					return false
				}
				if d.Salary == nil || *d.Salary != 50000.50 {
					fmt.Printf("Salary: expected 50000.50, got %v\n", d.Salary)
					return false
				}
				if d.IsEmployee == nil || *d.IsEmployee != true {
					fmt.Printf("IsEmployee: expected true, got %v\n", d.IsEmployee)
					return false
				}

				return true
			},
		},
		{
			name: "Nil pointer fields",
			url:  "http://example.com?name=John&age=30",
			dest: func() any {
				return &struct {
					Name   *string  `json:"name"`
					Age    *int     `json:"age"`
					Salary *float64 `json:"salary"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					Name   *string  `json:"name"`
					Age    *int     `json:"age"`
					Salary *float64 `json:"salary"`
				})

				if d.Name == nil || *d.Name != "John" {
					fmt.Printf("Name: expected 'John', got %v\n", d.Name)
					return false
				}
				if d.Age == nil || *d.Age != 30 {
					fmt.Printf("Age: expected 30, got %v\n", d.Age)
					return false
				}
				if d.Salary != nil {
					fmt.Printf("Salary: expected nil, got %v\n", d.Salary)
					return false
				}

				return true
			},
		},
		{
			name: "Slice of pointers",
			url:  "http://example.com?scores=90&scores=85&scores=95",
			dest: func() any {
				return &struct {
					Scores []*int `json:"scores"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					Scores []*int `json:"scores"`
				})
				expected := []int{90, 85, 95}
				if len(d.Scores) != len(expected) {
					return false
				}
				for i, v := range d.Scores {
					if *v != expected[i] {
						return false
					}
				}
				return true
			},
		},
		{
			name: "Empty values -- pointers",
			url:  "http://example.com?name=&age=&active=",
			dest: func() any {
				return &struct {
					Name   *string `json:"name"`
					Age    *int    `json:"age"`
					Active *bool   `json:"active"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					Name   *string `json:"name"`
					Age    *int    `json:"age"`
					Active *bool   `json:"active"`
				})
				fmt.Printf("Name: %v, Age: %v, Active: %v\n", d.Name, d.Age, d.Active)
				return d.Name == nil && d.Age == nil && d.Active == nil
			},
		},
		{
			name: "Empty values -- non-pointers",
			url:  "http://example.com?name=&age=&active=",
			dest: func() any {
				return &struct {
					Name   string `json:"name"`
					Age    int    `json:"age"`
					Active bool   `json:"active"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					Name   string `json:"name"`
					Age    int    `json:"age"`
					Active bool   `json:"active"`
				})
				fmt.Printf("Name: %v, Age: %v, Active: %v\n", d.Name, d.Age, d.Active)
				return d.Name == "" && d.Age == 0 && d.Active == false
			},
		},
		{
			name: "Type mismatch",
			url:  "http://example.com?age=notanumber",
			dest: func() any {
				return &struct {
					Age int `json:"age"`
				}{}
			},
			shouldFail: true,
		},
		{
			name: "Nested structs",
			url:  "http://example.com?name=John&address.city=NewYork&address.zip=10001",
			dest: func() any {
				return &struct {
					Name    string `json:"name"`
					Address struct {
						City string `json:"city"`
						Zip  int    `json:"zip"`
					} `json:"address"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					Name    string `json:"name"`
					Address struct {
						City string `json:"city"`
						Zip  int    `json:"zip"`
					} `json:"address"`
				})
				return d.Name == "John" && d.Address.City == "NewYork" && d.Address.Zip == 10001
			},
		},
		{
			name: "Double nested structs",
			url:  "http://example.com?name=John&address.city=NewYork&address.zip=10001&address.location.lat=40.7128&address.location.lng=-74.0060",
			dest: func() any {
				return &struct {
					Name    string `json:"name"`
					Address struct {
						City     string `json:"city"`
						Zip      int    `json:"zip"`
						Location struct {
							Lat float64 `json:"lat"`
							Lng float64 `json:"lng"`
						} `json:"location"`
					} `json:"address"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					Name    string `json:"name"`
					Address struct {
						City     string `json:"city"`
						Zip      int    `json:"zip"`
						Location struct {
							Lat float64 `json:"lat"`
							Lng float64 `json:"lng"`
						} `json:"location"`
					} `json:"address"`
				})
				return d.Name == "John" && d.Address.City == "NewYork" && d.Address.Zip == 10001 &&
					d.Address.Location.Lat == 40.7128 && d.Address.Location.Lng == -74.0060
			},
		},
		{
			name: "Nested struct with slice",
			url:  "http://example.com?name=John&address.city=NewYork&address.zip=10001&address.phones=1234567890&address.phones=0987654321",
			dest: func() any {
				return &struct {
					Name    string `json:"name"`
					Address struct {
						City   string   `json:"city"`
						Zip    int      `json:"zip"`
						Phones []string `json:"phones"`
					} `json:"address"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					Name    string `json:"name"`
					Address struct {
						City   string   `json:"city"`
						Zip    int      `json:"zip"`
						Phones []string `json:"phones"`
					} `json:"address"`
				})
				return d.Name == "John" && d.Address.City == "NewYork" && d.Address.Zip == 10001 &&
					reflect.DeepEqual(d.Address.Phones, []string{"1234567890", "0987654321"})
			},
		},
		{
			name: "Embedded structs",
			url:  "http://example.com?embeddedField=embeddedValue",
			dest: func() any {
				return &struct {
					Embedded
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					Embedded
				})
				fmt.Printf("Embedded: %+v\n", d.Embedded)
				return d.Embedded.EmbeddedField == "embeddedValue"
			},
		},
		{
			name: "Double embedded structs",
			url:  "http://example.com?embeddedField=embeddedValue&embeddedField2=embeddedValue2",
			dest: func() any {
				return &DoubleEmbedded{}
			},
			check: func(i any) bool {
				d := i.(*DoubleEmbedded)
				return d.Embedded.EmbeddedField == "embeddedValue" && d.EmbeddedField2 == "embeddedValue2"
			},
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
