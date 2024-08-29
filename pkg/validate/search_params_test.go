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

		//////////////////////////////////////////////////////////////////
		// MAPS
		//////////////////////////////////////////////////////////////////
		{
			name: "Basic map",
			url:  "http://example.com?data.key1=value1&data.key2=value2",
			dest: func() any {
				return &struct {
					Data map[string]string `json:"data"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					Data map[string]string `json:"data"`
				})
				return d.Data["key1"] == "value1" && d.Data["key2"] == "value2"
			},
		},
		{
			name: "Map with slice values",
			url:  "http://example.com?data.tags=go&data.tags=programming&data.scores=85&data.scores=90",
			dest: func() any {
				return &struct {
					Data map[string][]string `json:"data"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					Data map[string][]string `json:"data"`
				})
				return reflect.DeepEqual(d.Data["tags"], []string{"go", "programming"}) &&
					reflect.DeepEqual(d.Data["scores"], []string{"85", "90"})
			},
		},
		{
			name: "Empty map",
			url:  "http://example.com",
			dest: func() any {
				return &struct {
					Data map[string]string `json:"data"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					Data map[string]string `json:"data"`
				})
				return len(d.Data) == 0
			},
		},
		{
			name: "Map with empty values",
			url:  "http://example.com?data.key1=&data.key2=",
			dest: func() any {
				return &struct {
					Data map[string]string `json:"data"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					Data map[string]string `json:"data"`
				})
				return d.Data["key1"] == "" && d.Data["key2"] == ""
			},
		},
		{
			name: "Map with pointer values",
			url:  "http://example.com?data.name=John&data.age=30&data.active=true",
			dest: func() any {
				return &struct {
					Data map[string]*string `json:"data"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					Data map[string]*string `json:"data"`
				})
				return *d.Data["name"] == "John" && *d.Data["age"] == "30" && *d.Data["active"] == "true"
			},
		},
		{
			name: "Struct with multiple maps of different value types",
			url:  "http://example.com?stringMap.key1=value1&intMap.key2=42&boolMap.key3=true",
			dest: func() any {
				return &struct {
					StringMap map[string]string `json:"stringMap"`
					IntMap    map[string]int    `json:"intMap"`
					BoolMap   map[string]bool   `json:"boolMap"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					StringMap map[string]string `json:"stringMap"`
					IntMap    map[string]int    `json:"intMap"`
					BoolMap   map[string]bool   `json:"boolMap"`
				})
				return d.StringMap["key1"] == "value1" && d.IntMap["key2"] == 42 && d.BoolMap["key3"] == true
			},
		},
		{
			name: "Map with invalid type conversion",
			url:  "http://example.com?data.key=notanumber",
			dest: func() any {
				return &struct {
					Data map[string]int `json:"data"`
				}{}
			},
			shouldFail: true,
		},
		{
			name: "Map with mixed valid and invalid values",
			url:  "http://example.com?data.valid=42&data.invalid=notanumber",
			dest: func() any {
				return &struct {
					Data map[string]int `json:"data"`
				}{}
			},
			shouldFail: true,
		},
		{
			name: "Map key with dot", // Nested maps are not supported in the way structs are
			url:  "http://example.com?data.key.with.dot=value",
			dest: func() any {
				return &struct {
					Data map[string]string `json:"data"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					Data map[string]string `json:"data"`
				})
				return d.Data["key.with.dot"] == "value"
			},
		},

		//////////////////////////////////////////////////////////////////
		// POINTERS TO COMPLEX TYPES
		//////////////////////////////////////////////////////////////////
		{
			name: "Basic map pointer",
			url:  "http://example.com?data.key1=value1&data.key2=value2",
			dest: func() any {
				return &struct {
					Data *map[string]string `json:"data"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					Data *map[string]string `json:"data"`
				})
				return (*d.Data)["key1"] == "value1" && (*d.Data)["key2"] == "value2"
			},
		},
		{
			name: "Basic struct pointer",
			url:  "http://example.com?data.key1=value1&data.key2=value2",
			dest: func() any {
				return &struct {
					Data *struct {
						Key1 string `json:"key1"`
						Key2 string `json:"key2"`
					} `json:"data"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					Data *struct {
						Key1 string `json:"key1"`
						Key2 string `json:"key2"`
					} `json:"data"`
				})
				return d.Data.Key1 == "value1" && d.Data.Key2 == "value2"
			},
		},
		{
			name: "Basic slice pointer",
			url:  "http://example.com?data=value1&data=value2",
			dest: func() any {
				return &struct {
					Data *[]string `json:"data"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					Data *[]string `json:"data"`
				})
				if len(*d.Data) != 2 {
					fmt.Printf("Expected 2 elements, got %v\n", len(*d.Data))
					return false
				}
				if (*d.Data)[0] != "value1" || (*d.Data)[1] != "value2" {
					fmt.Printf("Expected [value1 value2], got %v\n", *d.Data)
					return false
				}
				return true
			},
		},

		//////////////////////////////////////////////////////////////////
		// MISC
		//////////////////////////////////////////////////////////////////
		{
			name: "Unsupported type",
			url:  "http://example.com?chanValue=something",
			dest: func() any {
				return &struct {
					ChanField chan int `json:"chanValue"`
				}{}
			},
			shouldFail: true,
		},
		{
			name: "Triple nested structs",
			url:  "http://example.com?level1.level2.level3.field=value",
			dest: func() any {
				return &struct {
					Level1 struct {
						Level2 struct {
							Level3 struct {
								Field string `json:"field"`
							} `json:"level3"`
						} `json:"level2"`
					} `json:"level1"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					Level1 struct {
						Level2 struct {
							Level3 struct {
								Field string `json:"field"`
							} `json:"level3"`
						} `json:"level2"`
					} `json:"level1"`
				})
				return d.Level1.Level2.Level3.Field == "value"
			},
		},
		{
			name: "Doubled nested structs with mixed pointers",
			url:  "http://example.com?level1.level2.field=value",
			dest: func() any {
				return &struct {
					Level1 struct {
						Level2 *struct {
							Field string `json:"field"`
						} `json:"level2"`
					} `json:"level1"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					Level1 struct {
						Level2 *struct {
							Field string `json:"field"`
						} `json:"level2"`
					} `json:"level1"`
				})
				return d.Level1.Level2.Field == "value"
			},
		},
		{
			name: "Empty query parameters",
			// GOAL:
			// Primitive types should be nil for pointers and zero values for non-pointers
			// Complex types will be initialized to their zero values no matter what
			url: "http://example.com?name_ptr=&name=&age_ptr=&age=&tags_ptr=&tags=&someStruct_ptr=&someStruct=&someMap_ptr=&someMap=",
			dest: func() any {
				return &struct {
					NamePtr       *string   `json:"name_ptr"`
					Name          string    `json:"name"`
					AgePtr        *int      `json:"age_ptr"`
					Age           int       `json:"age"`
					IsFunPtr      *bool     `json:"isFun_ptr"`
					IsFun         bool      `json:"isFun"`
					TagsPtr       *[]string `json:"tags_ptr"`
					Tags          []string  `json:"tags"`
					SomeStructPtr *struct {
						Field string `json:"field"`
					} `json:"someStruct_ptr"`
					SomeStruct struct {
						Field string `json:"field"`
					} `json:"someStruct"`
					SomeMapPtr *map[string]string `json:"someMap_ptr"`
					SomeMap    map[string]string  `json:"someMap"`
				}{}
			},
			check: func(i any) bool {
				d := i.(*struct {
					NamePtr       *string   `json:"name_ptr"`
					Name          string    `json:"name"`
					AgePtr        *int      `json:"age_ptr"`
					Age           int       `json:"age"`
					IsFunPtr      *bool     `json:"isFun_ptr"`
					IsFun         bool      `json:"isFun"`
					TagsPtr       *[]string `json:"tags_ptr"`
					Tags          []string  `json:"tags"`
					SomeStructPtr *struct {
						Field string `json:"field"`
					} `json:"someStruct_ptr"`
					SomeStruct struct {
						Field string `json:"field"`
					} `json:"someStruct"`
					SomeMapPtr *map[string]string `json:"someMap_ptr"`
					SomeMap    map[string]string  `json:"someMap"`
				})

				// Primitive types
				if d.NamePtr != nil {
					fmt.Printf("NamePtr: expected nil, got %v\n", d.NamePtr)
					return false
				}
				if d.Name != "" {
					fmt.Printf("Name: expected '', got %v\n", d.Name)
					return false
				}
				if d.AgePtr != nil {
					fmt.Printf("AgePtr: expected nil, got %v\n", d.AgePtr)
					return false
				}
				if d.Age != 0 {
					fmt.Printf("Age: expected 0, got %v\n", d.Age)
					return false
				}
				if d.IsFunPtr != nil {
					fmt.Printf("IsFunPtr: expected nil, got %v\n", d.IsFunPtr)
					return false
				}
				if d.IsFun != false {
					fmt.Printf("IsFun: expected false, got %v\n", d.IsFun)
					return false
				}

				// Complex types
				if d.TagsPtr == nil || len(*d.TagsPtr) != 0 {
					fmt.Printf("TagsPtr: got %v\n", d.TagsPtr)
					return false
				}
				if len(d.Tags) != 0 {
					fmt.Printf("Tags: got %v\n", d.Tags)
					return false
				}
				if d.SomeStructPtr == nil || d.SomeStructPtr.Field != "" {
					fmt.Printf("SomeStructPtr: got %v\n", d.SomeStructPtr)
					return false
				}
				if d.SomeStruct.Field != "" {
					fmt.Printf("SomeStruct: got %v\n", d.SomeStruct)
					return false
				}
				if d.SomeMapPtr == nil || len(*d.SomeMapPtr) != 0 {
					fmt.Printf("SomeMapPtr: got %v\n", d.SomeMapPtr)
					return false
				}
				if len(d.SomeMap) != 0 {
					fmt.Printf("SomeMap: got %v\n", d.SomeMap)
					return false
				}
				return true
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
