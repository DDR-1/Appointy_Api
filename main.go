package main

import (
	"encoding/json"
	"net/http"
	"regexp"

	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func GetMongoDbConnection() (*mongo.Client, error) {

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))

	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(context.Background(), readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}

	return client, nil
}

func getMongoDbCollection(DbName string, CollectionName string) (*mongo.Collection, error) {
	client, err := GetMongoDbConnection()

	if err != nil {
		return nil, err
	}

	collection := client.Database(DbName).Collection(CollectionName)

	return collection, nil
}


var (
    getUserRe    = regexp.MustCompile(`^\/users\/(\w+)$`)
    getPostRe    = regexp.MustCompile(`^\/posts\/(\w+)$`)
    getUserPostRe    = regexp.MustCompile(`^\/posts\/users\/(\w+)$`)
    createUserRe = regexp.MustCompile(`^\/users[\/]*$`)
    createPostRe = regexp.MustCompile(`^\/posts[\/]*$`)
)

const dbName = "appointydb"
const usercollectionName = "user"
const postcollectionName = "post"
const port = 800

type user struct {
    _id string `json:"id"`
    Name string `json:"name"`
    Email   string `json:"email"`
    Password string `json:"password"`
    Posts []string `json:"posts"`
}

type post struct {
    _id   string `json:"id"`
    Caption   string `json:"caption"`
    Url   string `json:"url"`
    Timestamp   string `json:"timestamp"`
}

type userHandler struct {
}

func (h *userHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("content-type", "application/json")
    switch {
    case r.Method == http.MethodGet && getUserRe.MatchString(r.URL.Path):
        h.GetUser(w, r)
        return
    case r.Method == http.MethodGet && getPostRe.MatchString(r.URL.Path):
        h.GetPost(w, r)
        return
    case r.Method == http.MethodPost && createUserRe.MatchString(r.URL.Path):
        h.CreateUser(w, r)
        return
    case r.Method == http.MethodPost && createPostRe.MatchString(r.URL.Path):
        h.CreatePost(w, r)
        return
    case r.Method == http.MethodGet && getUserPostRe.MatchString(r.URL.Path):
        h.GetUserPost(w, r)
        return
    default:
        notFound(w, r)
        return
    }
}

func (h *userHandler) GetUser(w http.ResponseWriter, r *http.Request) {
    matches := getUserRe.FindStringSubmatch(r.URL.Path)
    collection, err := getMongoDbCollection(dbName, usercollectionName)
	if err != nil {
		internalServerError(w, r)
		return
	}
    id,_ :=  primitive.ObjectIDFromHex(matches[1])

    var result user
    err = collection.FindOne(context.TODO(),bson.M{"_id": id}).Decode(&result)
    jsonBytes, err := json.Marshal(result)
    if err != nil {
        internalServerError(w, r)
        return
    }
    w.WriteHeader(http.StatusOK)
    w.Write(jsonBytes)
}

func (h *userHandler) GetPost(w http.ResponseWriter, r *http.Request) {
    matches := getPostRe.FindStringSubmatch(r.URL.Path)
    collection, err := getMongoDbCollection(dbName, postcollectionName)
	if err != nil {
		internalServerError(w, r)
		return
	}
    id,_ :=  primitive.ObjectIDFromHex(string(matches[1]))

    var result post
    err = collection.FindOne(context.TODO(),bson.M{"_id": id}).Decode(&result)
    jsonBytes, err := json.Marshal(result)
    if err != nil {
        internalServerError(w, r)
        return
    }
    w.WriteHeader(http.StatusOK)
    w.Write(jsonBytes)
}

func (h *userHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    collection, err := getMongoDbCollection(dbName, usercollectionName)
	if err != nil {
		internalServerError(w, r)
		return
	}

    var u user

    json.NewDecoder(r.Body).Decode(&u)

	res, err := collection.InsertOne(context.Background(), u)
    if err != nil {
        internalServerError(w, r)
        return
    }
    response, _ := json.Marshal(res)
    w.Write([]byte(response))
}

func (h *userHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
    collection, err := getMongoDbCollection(dbName, postcollectionName)
	if err != nil {
		internalServerError(w, r)
		return
	}

    var p post

    json.NewDecoder(r.Body).Decode(&p)

	res, err := collection.InsertOne(context.Background(), p)
    if err != nil {
        internalServerError(w, r)
        return
    }
    response, _ := json.Marshal(res)
    w.Write([]byte(response))
}

func (h *userHandler) GetUserPost(w http.ResponseWriter, r *http.Request) {
    matches := getUserPostRe.FindStringSubmatch(r.URL.Path)
    collection, err := getMongoDbCollection(dbName, usercollectionName)
    pcollection, err := getMongoDbCollection(dbName, postcollectionName)
	if err != nil {
		internalServerError(w, r)
		return
	}
    id,_ :=  primitive.ObjectIDFromHex(matches[1])

    var result user
    err = collection.FindOne(context.TODO(),bson.M{"_id": id}).Decode(&result)
    if err != nil {
        internalServerError(w, r)
        return
    }
    
    
    for i := 0; i < len(result.Posts); i++{
        id := result.Posts[i]
        ids, err := primitive.ObjectIDFromHex(string(id))
        var p post
        err = pcollection.FindOne(context.TODO(),bson.M{"_id": ids}).Decode(&p)
        if err != nil {
            internalServerError(w, r)
            return
        }
        jsonBytes, err := json.Marshal(p)
        if err != nil {
            internalServerError(w, r)
            return
        }
        w.Write([]byte(jsonBytes))
    }

}

func internalServerError(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusInternalServerError)
    w.Write([]byte("internal server error"))
}

func notFound(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusNotFound)
    w.Write([]byte("not found"))
}

func main() {
    mux := http.NewServeMux()
    userH := &userHandler{
    }
    mux.Handle("/users/", userH)
    mux.Handle("/posts/", userH)

    http.ListenAndServe("localhost:8080", mux)
}
