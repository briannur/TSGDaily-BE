package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	// Import dependencies
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Define the User struct
type User struct { 
	ID     		int    `gorm:"primaryKey;autoIncrement"`
	Username	string `gorm:"column:username"`
	Email		string `gorm:"column:email"`
	Password	string `gorm:"column:password"`
}

type QueryResult struct {
    User struct {
        Password string `json:"password"`
    } `json:"user"`
}

var schema graphql.Schema

// Initialize the GraphQL schema
func init() {
	var err error

	// Define the User type
	userType := graphql.NewObject(graphql.ObjectConfig{
		Name: "User",
		Fields: graphql.Fields{
			"id":       &graphql.Field{Type: graphql.Int},
			"username": &graphql.Field{Type: graphql.String},
			"email":    &graphql.Field{Type: graphql.String},
			"password": &graphql.Field{Type: graphql.String},
		},
	})

	// Define the root query type
	rootQuery := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"user": &graphql.Field{
				Type: userType,
				Args: graphql.FieldConfigArgument{
					"username": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"email": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					// Retrieve the GORM DB connection from the context
					db, ok := p.Context.Value("db").(*gorm.DB)
					if !ok {
						// If the DB connection is not found in the context, print an error message and return nil
						fmt.Println("db is not found in context")
						return nil, nil
					} else if db == nil {
						// If the DB connection is nil, print an error message and return nil
						fmt.Println("db is nil")
						return nil, nil
					}
					
					// Extract the username and email arguments from the query
					username, _ := p.Args["username"].(string)
					email, _ := p.Args["email"].(string)
					
					// Query the database for the first User struct that matches the given username or email
					var user User
					if err := db.Where("username = ? OR email = ?", username, email).First(&user).Error; err != nil {
						return nil, err
					}
					
					// Return the resulting User struct
					return user, nil
				},
			},
		},
	})

	// Define the schema
	schema, err = graphql.NewSchema(graphql.SchemaConfig{
		Query: rootQuery,
	})
	if err != nil {
		panic(err)
	}
}

func setDBContext(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx := context.WithValue(c.Request.Context(), "db", db)
        c.Request = c.Request.WithContext(ctx)
        c.Next()
    }
}

func main() {
    dsn := "host=localhost user=postgres password=root dbname=coba_sosmed port=5432 sslmode=disable"

    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        panic("failed to connect database")
    }

    // Define the query
    query := `
        query {
            user(username: "testuser", email: "") {
                password
            }
        }
    `

    c := context.WithValue(context.Background(), "db", db)

    // Execute the query
    result := graphql.Do(graphql.Params{
        Schema:   schema,
        RequestString: query,
        Context:  c,
    })

    if len(result.Errors) > 0 {
        log.Fatal(result.Errors)
    }

    // Unmarshal the result into the QueryResult struct
    var queryResult QueryResult
    resultBytes, _ := json.Marshal(result.Data)
    json.Unmarshal(resultBytes, &queryResult)

    // Put the password to variable and print
    password := queryResult.User.Password
	fmt.Println(password)

    r := gin.Default()

    r.Use(setDBContext(db))

    h := handler.New(&handler.Config{
        Schema: &schema,
        Pretty: true,
    })

    r.POST("/graphql", gin.WrapH(h))

    r.GET("/playground", gin.WrapH(playground.Handler("GraphQL playground", "/graphql")))

    if err := r.Run(":8080"); err != nil {
        panic(err)
    }
}
