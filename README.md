## jsparser

jsparser is a json parser for GO. It is efficient to parse large json data with streaming fashion.

### Usage
```json
{
  "books": [{
      "title": "The Iliad and The Odyssey",
      "price": 12.95,
      "comments": [{
        "rating": 4,
        "comment": "Best translation I've read."
      }, {
        "rating": 2,
        "comment": "I like other versions better."
      }]
    },
    {
      "title": "Anthology of World Literature",
      "price": 24.95,
      "comments": [{
        "rating": 4,
        "comment": "Excellent overview of world literature."
      }, {
        "rating": 3,
        "comment": "Needs more modern literature."
      }]
    }
  ]
}
```
<b>Stream</b> over books

```go
f, _ := os.Open("input.json")
br := bufio.NewReaderSize(f,65536)
parser := jsparser.NewJSONParser(br, "books")

for json:= range parser.Stream() {
    fmt.Println(json.ObjectVals["title"].StringVal)
    fmt.Println(json.ObjectVals["price"].StringVal)
    fmt.Println(json.ObjectVals["comments"].ArrayVals[0].ObjectVals["rating"].StringVal)
}
  
```

<b>Skip</b> props for efficiency

```go
parser := pr.NewJSONParser(br, "books").SkipProps([]string{"comments", "price"})  
```

<b>Error</b> handling

```go
for json:= range parser.Stream() {
    if json.Err !=nil {
      // handle error
    }
}
```

<b>Progress</b> of parsing
```go
// total byte read to calculate the progress of parsing
parser.TotalReadSize
```


If you are interested check also [xml parser](https://github.com/tamerh/xml-stream-parser) which works similarly.
