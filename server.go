package main

import (
    "fmt"
    "log"
    "strings"
    "flag"
    "net/http"
    "github.com/dbalduini/encurtador/url"
    "encoding/json"

)

var (
    porta *int
    logLigado *bool
    urlBase string
)

func init() {
    porta = flag.Int("p", 8888, "porta")
    logLigado = flag.Bool("l", true, "log ligado/desligado")
    flag.Parse()

    urlBase = fmt.Sprintf("http://localhost:%d", *porta)
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

type Redirecionador struct {
    stats chan string
}

func (red *Redirecionador) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    caminho := strings.Split(r.URL.Path, "/")
    id := caminho[len(caminho)-1]

    if url := url.Buscar(id); url != nil {
        http.Redirect(w, r, url.Destino, http.StatusMovedPermanently)
        red.stats <- id
    } else {
        http.NotFound(w, r)
    }
}

func Visualizador(w http.ResponseWriter, r *http.Request) {
    caminho := strings.Split(r.URL.Path, "/")
    id := caminho[len(caminho)-1]

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
    logar("Registering stats...")
    for id := range stats {
        url.RegistrarClick(id)
        logar("Click de log registrado para %s.", id)
    }
}

func logar(formato string, valores ...interface{}) {
    if *logLigado {
        log.Printf(fmt.Sprintf("%s\n", formato), valores...)
    }
}

func main() {
    url.SetRepositorio(url.NovoRepositorioMemoria())

    p := *porta

    stats := make(chan string)
    defer close(stats)
    go registrarEstatisticas(stats)

    http.HandleFunc("/api/encurtar", Encurtador)
    http.HandleFunc("/api/stats/", Visualizador)
    http.Handle("/r/", &Redirecionador{stats})

    logar("Iniciando servidor na porta %d...", p)
    log.Fatal(http.ListenAndServe(
        fmt.Sprintf(":%d", p), nil))
}