package main

import (
{{ range $key, $_ := .Modules }}    "{{ $key }}"
{{end}})

func main() {
    resp, err := http.Get({{ .Url }})
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Fatal(err)
    }
    log.Print(string(body))
}
