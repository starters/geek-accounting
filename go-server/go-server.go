package server

import (
	"encoding/json"
	//"io/ioutil"
	"log"
    "net/http"
    "github.com/gorilla/mux"
    "github.com/mcesarhm/geek-accounting/go-server/domain"

    "appengine"
)

const PathPrefix = "/charts-of-accounts"

func init() {
	r := mux.NewRouter()
    r.HandleFunc(PathPrefix, getAllHandler(domain.AllChartsOfAccounts)).Methods("GET")
    r.HandleFunc(PathPrefix, postHandler(NewChartOfAccounts)).Methods("POST")
    /*
    r.HandleFunc(PathPrefix+"/{id}", errorHandler(GetTask)).Methods("GET")
    r.HandleFunc(PathPrefix+"/{id}", errorHandler(UpdateTask)).Methods("PUT")
    */
    http.Handle("/", r)
}

func NewChartOfAccounts(c appengine.Context, m map[string]interface{}) (coa interface{}, err error) {
    coa = &domain.ChartOfAccounts{Name: m["name"].(string)}
    err = domain.SaveChartOfAccounts(c, coa.(*domain.ChartOfAccounts))
    return
}

func getAllHandler(f func(appengine.Context) (interface{}, error)) http.HandlerFunc {
	return errorHandler(func(w http.ResponseWriter, r *http.Request) error {
		items, err := f(appengine.NewContext(r))
		if err != nil {
			return err
		}
		return json.NewEncoder(w).Encode(items)
	})
}

func postHandler(f func(appengine.Context, map[string]interface{}) (interface{}, error)) http.HandlerFunc {
	return errorHandler(func(w http.ResponseWriter, r *http.Request) error {
		/*
		b, _ := ioutil.ReadAll(r.Body)
		log.Println(string(b))
		*/
		var req interface{}
	    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
	        return badRequest{err}
	    }

	    m := req.(map[string]interface{})
	    item, err := f(appengine.NewContext(r), m)
	    if err != nil {
	        return badRequest{err}
	    }

	    json.NewEncoder(w).Encode(item)

	    return nil
	})
}


type badRequest struct{ error }

type notFound struct{ error }

// errorHandler wraps a function returning an error by handling the error and returning a http.Handler.
// If the error is of the one of the types defined above, it is handled as described for every type.
// If the error is of another type, it is considered as an internal error and its message is logged.
func errorHandler(f func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        err := f(w, r)
        if err == nil {
			return
        }
        switch err.(type) {
        case badRequest:
            http.Error(w, "Error: " + err.Error(), http.StatusBadRequest)
        case notFound:
            http.Error(w, "Error: item not found", http.StatusNotFound)
        default:
            log.Println(err)
            http.Error(w, "Internal error", http.StatusInternalServerError)
        }
    }
}
