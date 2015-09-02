package main

import (
    "fmt"
    "log"
    "net/http"
    "strings"
    "github.com/dbalduini/encurtador/url"
    "encoding/json"
)

var (
    porta int
    urlBase string
    stats chan string
)

func init() {
    porta = 8888
    urlBase = fmt.Sprintf("http://localhost:%d", porta)
}

type Headers map[string]string

func Encurtador(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        responderCom(w, http.StatusMethodNotAllowed, Headers {"Allow": "POST"})
        return
    }
    
    url, nova, err := url.BuscarOuCriarNovaUrl(extrairUrl(r))

    if err != nil {
        responderCom(w, http.StatusBadRequest, nil)
        return
    }

    var status int
    if nova {
        status = http.StatusCreated
    } else {
        status = http.StatusOK
    }

    urlCurta := fmt.Sprintf("%s/r/%s", urlBase, url.Id)
    responderCom(w, status, Headers{
        "Location": urlCurta,
        "Link": fmt.Sprintf("<%s/api/stats/%s>; rel=\"stats\"", urlBase, url.Id),
    })

}

func Redirecionador(w http.ResponseWriter, r *http.Request) {
    caminho := strings.Split(r.URL.Path, "/")
    id := caminho[len(caminho)-1]

    if url := url.Buscar(id); url != nil {
        http.Redirect(w, r, url.Destino, 
            http.StatusMovedPermanently)
        
        fmt.Println("Putting id into stats")
        stats <- id
        fmt.Println("Done")
    } else {
        http.NotFound(w, r)
    }
}

func Visualizador(w http.ResponseWriter, r *http.Request) {
    fmt.Println("AQUI")
    caminho := strings.Split(r.URL.Path, "/")
    id := caminho[len(caminho)-1]
    fmt.Println(id)

    if url := url.Buscar(id); url != nil {
        json, err := json.Marshal(url.Stats())

        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            return
        }

        responderComJson(w, string(json))
    } else {
        http.NotFound(w, r)
    }
}

func responderCom(w http.ResponseWriter, status int, headers Headers) {
    for k, v := range headers {
        w.Header().Set(k, v)
    }
    w.WriteHeader(status)
}

func responderComJson(w http.ResponseWriter, resposta string) {
    responderCom(w, http.StatusOK, Headers{
        "Content-Type": "application/json",
    })
    fmt.Fprintf(w, resposta)
}

func extrairUrl(r *http.Request) string {
    url := make([]byte, r.ContentLength, r.ContentLength)
    r.Body.Read(url)
    return string(url)
}

func registrarEstatisticas(stats <-chan string) {
    fmt.Println("Registering stats...")
    for id := range stats {
        url.RegistrarClick(id)
    }
}

func main() {
    url.SetRepositorio(url.NovoRepositorioMemoria())

    stats := make(chan string)
    defer close(stats)
    go registrarEstatisticas(stats)

    http.HandleFunc("/api/encurtar", Encurtador)
    http.HandleFunc("/api/stats/", Visualizador)
    http.HandleFunc("/r/", Redirecionador)

    log.Fatal(http.ListenAndServe(
        fmt.Sprintf(":%d", porta), nil))
}