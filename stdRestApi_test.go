package elorm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestStdRestApi(t *testing.T) {

	f := mockFactory()
	ordersDef := mockEntityDef_Orders(f)

	mock_ClearEntities(t)
	refs := mock_SeedEntities(t)

	router := http.NewServeMux()

	createEntity := func() (*Entity, error) {
		return f.CreateEntity(ordersDef)
	}

	shopsRestApiConfig := CreateStdRestApiConfig(
		ordersDef,
		f.LoadEntity,
		ordersDef.SelectEntities,
		createEntity)
	shopsRestApiConfig.DefaultPageSize = 10
	shopsRestApiConfig.DefaultSorts = func(r *http.Request) ([]*SortItem, error) {
		return []*SortItem{{Field: ordersDef.FieldDefByName("OrderNbr"), Asc: true}}, nil
	}
	shopsRestApiConfig.AdditionalFilter = func(r *http.Request) ([]*Filter, error) {
		filters := []*Filter{AddFilterEQ(ordersDef.FieldDefByName("IsDeleted"), false)}
		return filters, nil
	}
	//shopsRestApiConfig.Context = LoadUserFromHttpToContext
	//shopsRestApiConfig.BeforeMiddleware = UserRequiredForEdit

	router.HandleFunc("/api/orders", HandleRestApi(shopsRestApiConfig))

	server := &http.Server{
		Addr:    "localhost:8081",
		Handler: router,
	}

	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			panic("Failed to start server: " + err.Error())
		}
	}()

	type reqLine struct {
		method string
		body   string
	}

	newOrder, err := f.CreateEntity(ordersDef)
	if err != nil {
		t.Fatalf("Failed to create new order entity: %v", err)
	}

	var newBuf bytes.Buffer
	err = json.NewEncoder(&newBuf).Encode(newOrder)
	if err != nil {
		t.Fatalf("Failed to encode new order entity: %v", err)
	}

	order9, err := f.LoadEntity(refs[9])
	if err != nil {
		t.Fatalf("Failed to load order entity: %v", err)
	}
	var putBuf bytes.Buffer
	err = json.NewEncoder(&putBuf).Encode(order9)
	if err != nil {
		t.Fatalf("Failed to encode order entity for PUT: %v", err)
	}

	order9.RefString()

	urls := map[string]reqLine{
		"http://localhost:8081/api/orders":                                         {method: "GET", body: ""},
		"http://localhost:8081/api/orders?page=2&pagesize=5":                       {method: "GET", body: ""},
		"http://localhost:8081/api/orders?page=2&sortby=Ref":                       {method: "GET", body: ""},
		"http://localhost:8081/api/orders?page=2&sortby=Ref%20desc":                {method: "GET", body: ""},
		"http://localhost:8081/api/orders?page=2&sortby=Ref%20desc&OrderNbr=123":   {method: "GET", body: ""},
		fmt.Sprintf("http://localhost:8081/api/orders?ref=%s", refs[seedCount-5]):  {method: "GET", body: ""},
		fmt.Sprintf("http://localhost:8081/api/orders?ref=%s", refs[seedCount-10]): {method: "DELETE", body: ""},
		fmt.Sprintf("http://localhost:8081/api/orders?ref=%s", refs[seedCount-15]): {method: "DELETE", body: ""},
		"http://localhost:8081/api/orders?q=1":                                     {method: "POST", body: newBuf.String()},
		fmt.Sprintf("http://localhost:8081/api/orders?ref=%s", order9.RefString()): {method: "PUT", body: putBuf.String()},
	}

	var r *http.Response
	for url, verb := range urls {

		switch verb.method {
		case "GET":
			r, err = http.Get(url)
			f.AggressiveReadingCache = !f.AggressiveReadingCache

		case "DELETE":
			var req *http.Request
			req, err = http.NewRequest(http.MethodDelete, url, nil)
			if err != nil {
				t.Fatalf("Failed to create DELETE request: %v", err)
			}
			r, err = http.DefaultClient.Do(req)
			ordersDef.UseSoftDelete = !ordersDef.UseSoftDelete

		case "POST":
			var req *http.Request
			req, err = http.NewRequest(http.MethodPost, url, nil)
			if err != nil {
				t.Fatalf("Failed to create POST request: %v", err)
			}
			req.Body = io.NopCloser(strings.NewReader(verb.body))
			r, err = http.DefaultClient.Do(req)

		case "PUT":
			var req *http.Request
			req, err = http.NewRequest(http.MethodPut, url, nil)
			if err != nil {
				t.Fatalf("Failed to create PUT request: %v", err)
			}
			req.Body = io.NopCloser(strings.NewReader(verb.body))
			r, err = http.DefaultClient.Do(req)
		}

		if err != nil {
			t.Fatalf("Failed to make GET request: %v", err)
		}
		if r.StatusCode < 200 || r.StatusCode > 299 {
			t.Fatalf("Expected status 2XX, got %d", r.StatusCode)
		}
	}

	err = server.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("Failed to shutdown server: %v", err)
	}

}
