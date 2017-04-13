package main

import (
    "encoding/json"
    "log"
    "net/http"
    "github.com/gorilla/mux"
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
)

type CompanyInfo struct {
    Id        bson.ObjectId `bson:"_id,omitempty"`
    Name      string
    ISIN     string `bson:"ISIN"`
    Displayname string `bson:"displayName"`
    SEDOL string `bson:"SEDOL,omitempty"`
    CIK int32 `bson:"CIK"`
    Symbol string
    Industry string `bson:"Industry"`
}

type Error struct {
    Message string `json:"message,omitempty"`
}

func Middleware(h http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header["Accesstoken"][0]
        channel := make(chan bool)
        go verifyToken(token, channel)
        connected := <- channel

        if connected == false  {
            ErrorWithJSON(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
        } else {
            h.ServeHTTP(w, r)
        }
    })}

func verifyToken(t string, c chan bool)  {
    client := &http.Client{}
    req, err := http.NewRequest("GET", "http://localhost:2989/verifyToken", nil)
    req.Header.Add("accesstoken", t)
    req.Header.Add("app", "insight")
    response, err := client.Do(req)
    if err != nil {
        log.Fatal(err)
        c <- false
    } else {
        defer response.Body.Close()
        if response.StatusCode == http.StatusOK {
            c <- true
        } else {
            c <- false
        }
    }
}

func ErrorWithJSON(w http.ResponseWriter, message string, code int) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(code)
    errorMessage := Error{message}
    json.NewEncoder(w).Encode(errorMessage)
}

func ResponseWithJSON(w http.ResponseWriter, json []byte, code int) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(code)
    w.Write(json)
}

func GetCompaniesEndpoint(s *mgo.Session) func(w http.ResponseWriter, req *http.Request) {
    return func(w http.ResponseWriter, req *http.Request) {
        session := s.Copy()
        defer session.Close()

        params := req.URL.Query()

        isin := string(params["isin"][0])

        c := session.DB("QikSayDB").C("companies")

        var result CompanyInfo

        err2 := c.Find(bson.M{"ISIN": isin}).One(&result)
        if err2 != nil {
            panic(err2)
        }

        respBody, err := json.Marshal(result)
        if err != nil {
            log.Fatal(err)
        }

        ResponseWithJSON(w, respBody, http.StatusOK)

    }
}

func main() {
    session, err := mgo.Dial("mongo1.truvaluelabs.com")
    if err != nil {
        panic(err)
    }
    defer session.Close()

    router := mux.NewRouter()
    router.Handle("/go/companies", Middleware(http.HandlerFunc(GetCompaniesEndpoint(session)))).Methods("GET")
    log.Fatal(http.ListenAndServe(":12345", router))
}
