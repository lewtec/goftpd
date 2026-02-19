package main

import (
	"flag"
	"log"
	"os"

	"github.com/lucasew/goftpd/internal/server"
	"github.com/lucasew/goftpd/internal/utils"
)

var (
	port     = flag.String("p", "80", "Onde eu vou escutar")
	basepath = flag.String("d", "./", "Root folder")
)

func main() {
	flag.Parse()

	// Verify directory exists before starting (preserves original behavior)
	if _, err := os.Stat(*basepath); os.IsNotExist(err) {
		utils.ReportError(err, "[main] erro: A pasta de trabalho não existe")
		flag.Usage()
		os.Exit(1)
	}

	srv := server.New(*basepath, *port)

	bindTo := "0.0.0.0:" + *port

	log.Println("[main] info: Iniciando servidor...")
	log.Printf("[main] info: Meu trabalho acontecerá na pasta: %s\n", *basepath)
	log.Printf("[main] info: Entendido, vamos trabalhar em %s\n", bindTo)
	log.Println("[main] info: Pau na máquina!")

	if err := srv.Start(); err != nil {
		utils.ReportError(err, "[main] erro")
		os.Exit(1)
	}
}
