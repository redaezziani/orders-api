package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Task struct represents a task document in MongoDB
type Task struct {
	ID        string    `json:"id,omitempty" bson:"_id,omitempty"`
	Name      string    `json:"name" bson:"name"`
	Completed bool      `json:"completed" bson:"completed"`
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
}

var client *mongo.Client

func main() {
	// Initialize the MongoDB client
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	var err error
	client, err = mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())

	// Initialize the router
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// Define API routes
	r.Get("/tasks", getTasks)
	r.Get("/tasks/{id}", getTask)
	r.Post("/tasks", createTask)
	r.Put("/tasks/{id}", updateTask)
	r.Delete("/tasks/{id}", deleteTask)

	// Start the server
	server := http.Server{
		Addr:    ":3000",
		Handler: r,
	}

	fmt.Println("Server is running on :3000")
	log.Fatal(server.ListenAndServe())
}

func getTasks(w http.ResponseWriter, r *http.Request) {
	collection := client.Database("mydb").Collection("tasks")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var tasks []Task
	cur, err := collection.Find(ctx, bson.M{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch tasks")
		return
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var task Task
		err := cur.Decode(&task)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to decode tasks")
			return
		}
		tasks = append(tasks, task)
	}

	writeJSON(w, http.StatusOK, tasks)
}

func getTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	collection := client.Database("mydb").Collection("tasks")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var task Task
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&task)
	if err != nil {
		writeError(w, http.StatusNotFound, "Task not found")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func createTask(w http.ResponseWriter, r *http.Request) {
	var task Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	task.CreatedAt = time.Now()

	collection := client.Database("mydb").Collection("tasks")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := collection.InsertOne(ctx, task)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create task")
		return
	}

	task.ID = result.InsertedID.(string)
	writeJSON(w, http.StatusCreated, task)
}

func updateTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var task Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	collection := client.Database("mydb").Collection("tasks")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = collection.ReplaceOne(ctx, bson.M{"_id": id}, task)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update task")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func deleteTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	collection := client.Database("mydb").Collection("tasks")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete task")
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	errorMessage := map[string]string{"error": message}
	json.NewEncoder(w).Encode(errorMessage)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
