# Strapi code generator for golang

This project is made up of a python script and support go code
to ease interacting with a Strapi server from golang by generating
code for all data structures and CRUD operations.

## Running code generator

Simply run

```sh
python3 ./strapi-gg.py https://my-strapi.com/api .token > strapi-generated.go
```

Replacing first argument with your strapi API endpoint, and second argument
with the name of a file containing your token.

Then add "strapi-common.go" and the generated file "strapi-generated.go" to your
project.


## API Usage

Let's suppose you have the following resources in Strapi:

    Book
      Title string
      Author Author
      Publisher Publisher
      InStock boolean
    Author
      Name string
      Residence Country
    Publisher
      Name string
    Country
      Name string

and created a `Strapi` object like so:

```go
s := &Strapi {
   Endpoint: "https://my-strapi.com/api",
   Token: "ABCABC...",
}
```

### Listing

Listing books is achieved by creating a `BookResponse` and passing it to
`Strapi.List` with filters and populate requests:

```go
books := BookResponse{}
err := s.List(&books, "*") // Populate all level 1 relations
err := s.List(&books, "publisher.id", "author.id") // Populate only ids of relations
err := s.List(&books, "author.country*", "publisher") // Populate level 2 author.country and level 1 publisher
err := s.List(&books, "*", "author.id=42") // Filter by author id
```

### Getting

Getting a single entry works like this:

```go
book := BookPtr{}
err := s.Get(&book, 42, "*")
```

### Updating

Two methods are provided: `Update` and `UpdateNullable`. The first one
will not delete relations that are missing in the struct, the second one will.

```go
book := BookPtr{}
err := s.Get(&book, 42) // Get without populating relations
book.Data.Attrs.InStock = True
err = s.Update(book.Data) // Will not touch relations

err = s.Get(&book, 42, "*")
book.Data.Attrs.Publisher = nil
err = s.UpdateNullable(book.Data) // Will delete publisher relation
```

### Creating

```go
bookAttrs := BookWriteAttrs {
    Title: "Best book you'll ever read",
    Publisher: 42,
    Author: 51,
}
id, err := s.Add(&bookAttrs)
```

### Deleting

```go
err = s.Delete(&someBook)
err = s.DeleteResource("books", 1234)
```