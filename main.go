package main

import (
	"html/template"
	"net/http"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

var client *redis.Client
var store = sessions.NewCookieStore([]byte("t0p-s3cr3t-c00k!3"))
var templates *template.Template

func main() {
	client = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	templates = template.Must(template.ParseGlob("templates/*.html"))
	r := mux.NewRouter()
	r.HandleFunc("/", indexGetHandler).Methods("GET")
	r.HandleFunc("/", indexPostHandler).Methods("POST")
	r.HandleFunc("/login", loginGetHandler).Methods("GET")
	r.HandleFunc("/login", loginPostHandler).Methods("POST")
	r.HandleFunc("/register", registerGetHandler).Methods("GET")
	r.HandleFunc("/register", registerPostHandler).Methods("POST")
	fs := http.FileServer(http.Dir("./static/"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
	http.Handle("/", r)
	http.ListenAndServe(":8080", nil)
}

func indexGetHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	_, ok := session.Values["username"]
	if !ok {
		http.Redirect(w, r, "/login", 302)
		return
	}
	comments, err := client.LRange("comments", 0, 10).Result()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}
	templates.ExecuteTemplate(w, "index.html", comments)
}

func indexPostHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	comment := r.PostForm.Get("comment")
	err := client.LPush("comments", comment).Err()
	// Error handling if comment don't get submitted to db
	// let user know the status in case of error
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}
	//client.LPush("comments", comment)
	http.Redirect(w, r, "/", 302)
}

func loginGetHandler(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "login.html", nil)
}

//  Post loginPostHandler
func loginPostHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.PostForm.Get("username")
	password := r.PostForm.Get("password")
	hash, err := client.Get("user:" + username).Bytes()
	if err == redis.Nil {
		templates.ExecuteTemplate(w, "login.html", "unknown user")
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}
	// if db return nil value for this user
	// user don't exist
	//if err != nil {
	//	return
	//}
	err = bcrypt.CompareHashAndPassword(hash, []byte(password))
	if err != nil {
		templates.ExecuteTemplate(w, "login.html", "invalid login credentials")
		return
	}
	// store session as cookie
	session, _ := store.Get(r, "session")
	session.Values["username"] = username
	session.Save(r, w)
	http.Redirect(w, r, "/", 302)
}

// Get registerGetHandler to open /register page
func registerGetHandler(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "register.html", nil)

}

// Post registerGetHandler to submit /register page
func registerPostHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.PostForm.Get("username")
	password := r.PostForm.Get("password")
	cost := bcrypt.DefaultCost
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}
	if err != nil {
		return
	}
	err = client.Set("user:"+username, hash, 0).Err()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}
	http.Redirect(w, r, "/login", 302)

}
