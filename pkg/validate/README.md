# Validate

## URLSearchParamsInto

`URLSearchParamsInto` is a method of the `Validate` struct that parses URL query parameters from an HTTP request into a destination struct and validates the resulting struct.

### Signature

```go
func (v Validate) URLSearchParamsInto(r *http.Request, destStructPtr any) error
```

### Parameters

- `r *http.Request`: The HTTP request containing the URL query parameters to be parsed.
- `destStructPtr any`: A pointer to the destination struct where the parsed values will be stored.

### Return Value

Returns an error if parsing or validation fails, otherwise returns `nil`.

### Usage Examples

#### Basic Types

```go
type BasicForm struct {
    Name    string  `json:"name"`
    Age     int     `json:"age"`
    Height  float64 `json:"height"`
    IsAdmin bool    `json:"isAdmin"`
}

// URL: /?name=John&age=30&height=1.75&isAdmin=true
func handleBasicForm(r *http.Request) {
    var form BasicForm
    err := v.URLSearchParamsInto(r, &form)
    if err != nil {
        // Handle error
    }
    // form is now populated with:
    // {Name: "John", Age: 30, Height: 1.75, IsAdmin: true}
}
```

#### Slices

```go
type SliceForm struct {
    Tags []string `json:"tags"`
    Scores []int  `json:"scores"`
}

// URL: /?tags=go&tags=programming&scores=85&scores=90&scores=95
func handleSliceForm(r *http.Request) {
    var form SliceForm
    err := v.URLSearchParamsInto(r, &form)
    if err != nil {
        // Handle error
    }
    // form is now populated with:
    // {Tags: ["go", "programming"], Scores: [85, 90, 95]}
}
```

#### Nested Structs

```go
type Address struct {
    Street string `json:"street"`
    City   string `json:"city"`
    ZIP    string `json:"zip"`
}

type UserForm struct {
    Name    string  `json:"name"`
    Address Address `json:"address"`
}

// URL: /?name=Alice&address.street=123 Main St&address.city=Anytown&address.zip=12345
func handleNestedForm(r *http.Request) {
    var form UserForm
    err := v.URLSearchParamsInto(r, &form)
    if err != nil {
        // Handle error
    }
    // form is now populated with:
    // {Name: "Alice", Address: {Street: "123 Main St", City: "Anytown", ZIP: "12345"}}
}
```

#### Maps

```go
type MapForm struct {
    Attributes map[string]string `json:"attributes"`
}

// URL: /?attributes.color=blue&attributes.size=large&attributes.material=cotton
func handleMapForm(r *http.Request) {
    var form MapForm
    err := v.URLSearchParamsInto(r, &form)
    if err != nil {
        // Handle error
    }
    // form is now populated with:
    // {Attributes: {"color": "blue", "size": "large", "material": "cotton"}}
}
```

#### Pointer Fields

```go
type PointerForm struct {
    Name     *string  `json:"name"`
    Age      *int     `json:"age"`
    IsActive *bool    `json:"isActive"`
}

// URL: /?name=Bob&age=25&isActive=true
func handlePointerForm(r *http.Request) {
    var form PointerForm
    err := v.URLSearchParamsInto(r, &form)
    if err != nil {
        // Handle error
    }
    // form is now populated with pointers to values:
    // {Name: &"Bob", Age: &25, IsActive: &true}
}
```

#### Mixed Types

```go
type ComplexForm struct {
    Name       string            `json:"name"`
    Age        int               `json:"age"`
    Hobbies    []string          `json:"hobbies"`
    Address    Address           `json:"address"`
    Attributes map[string]string `json:"attributes"`
    IsStudent  *bool             `json:"isStudent"`
}

// URL: /?name=Eve&age=22&hobbies=reading&hobbies=painting&address.city=Springfield&address.zip=67890&attributes.department=Art&attributes.year=2nd&isStudent=true
func handleComplexForm(r *http.Request) {
    var form ComplexForm
    err := v.URLSearchParamsInto(r, &form)
    if err != nil {
        // Handle error
    }
    // form is now populated with a mix of types
}
```

### Notes

- Empty values for non-pointer fields are ignored and leave the field unchanged.
- Pointer fields are set to nil for empty values in URL parameters.
- Multiple values with the same key are collected into slices.
- Nested struct fields are accessed using dot notation in URL parameters.
- Map keys use dot notation after the map field name, but don't support further nesting.
- The function uses `json` tags to map URL parameter names to struct fields, falling back to field names if no tag is present.
- Validation errors will be returned if the struct has validation tags and the parsed data doesn't meet the criteria.
