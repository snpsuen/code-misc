package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"runtime"
	"unsafe"

	"github.com/gorilla/mux"
)

/*

#include <stdlib.h>
#include <stdio.h>

void *foo(size_t count) {
        void *thing = (void *)malloc(sizeof(char) * count * 1024 * 1024);
        printf("count %zu\n", count);
        return thing;
}

*/
import "C"

type Memory struct {
	size   int
	memory unsafe.Pointer
}

func (m *Memory) alloc() {
	m.memory = C.foo(C.size_t(m.size))
	fmt.Printf("alloced memory %v\n", m.memory)
	runtime.SetFinalizer(m, free)
}

func free(m *Memory) {
	fmt.Printf("freeing memory %d %v\n", m.size, m.memory)
	C.free(unsafe.Pointer(m.memory))
}

var color string = "a multi-colored shine on all surfaces"

// var allocations map[string]bytes.Buffer
var allocations map[string]*Memory
var count int

func handleHealth(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, color)
}

func handleAllocate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	amount, err := strconv.Atoi(vars["amount"])
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintf(w, "Invalid argument, amount should be an integer")
	}

	count = count + 1

	/*
		buf := bytes.Buffer{}
		buf.Grow(amount)

		id := fmt.Sprintf("%d", count)
		allocations[id] = buf
	*/
	id := fmt.Sprintf("%d", count)
	m := Memory{}
	m.size = amount
	m.alloc()
	allocations[id] = &m
	fmt.Fprintf(w, id)
}

func handleDeallocate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if _, ok := allocations[id]; ok {
		delete(allocations, id)
		fmt.Fprintf(w, "OK")
		runtime.GC()
		return
	}

	w.WriteHeader(404)
	fmt.Fprintf(w, "not found")
}

func handleClear(w http.ResponseWriter, r *http.Request) {

	for key := range allocations {
		delete(allocations, key)
	}
	// debug.FreeOSMemory()
	runtime.GC()
	fmt.Fprintf(w, "OK")
}

func handleAllocations(w http.ResponseWriter, r *http.Request) {
	var total int
	for key, value := range allocations {
		total = total + value.size
		fmt.Fprintf(w, fmt.Sprintf("%s => %d\n", key, value.size))
	}
	fmt.Fprintf(w, fmt.Sprintf("total %d\n", total))
}

func main() {
	if len(os.Args) > 1 && os.Args[1] != "default" {
		color = strings.ToUpper(os.Args[1])
	}

	allocations = make(map[string]*Memory)

	r := mux.NewRouter()

	r.HandleFunc("/", handleRoot)
	r.HandleFunc("/health", handleHealth)
	r.HandleFunc("/allocate/{amount}", handleAllocate).Methods("GET")
	r.HandleFunc("/deallocate/{id}", handleDeallocate).Methods("GET")
	r.HandleFunc("/clear", handleClear)
	r.HandleFunc("/allocations", handleAllocations)
	http.Handle("/", r)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println(err)
	}
}
