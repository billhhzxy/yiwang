package main

import (
	"flag"
	"log"

	"github.com/gin-gonic/gin"

	"yiwang/internal/api"
	"yiwang/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dsn := flag.String("dsn", "root:123456@tcp(127.0.0.1:3306)/yiwang?parseTime=true&loc=Local", "MySQL DSN")
	flag.Parse()

	st, err := store.New(*dsn)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}

	r := gin.Default()
	api.New(st).Register(r.Group("/api"))
	r.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})
	r.StaticFile("/app.js", "./web/app.js")
	r.StaticFile("/styles.css", "./web/styles.css")

	log.Printf("listening on %s (MySQL DSN: %s)", *addr, *dsn)
	if err := r.Run(*addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
