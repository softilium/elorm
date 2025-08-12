package elorm

import (
	"context"
	"fmt"
	"net/http"
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

	urls := map[string]string{
		"http://localhost:8081/api/orders":                                         "GET",
		"http://localhost:8081/api/orders?page=2&pagesize=5":                       "GET",
		"http://localhost:8081/api/orders?page=2&sortby=Ref":                       "GET",
		"http://localhost:8081/api/orders?page=2&sortby=Ref%20desc":                "GET",
		"http://localhost:8081/api/orders?page=2&sortby=Ref%20desc&OrderNbr=123":   "GET",
		fmt.Sprintf("http://localhost:8081/api/orders?ref=%s", refs[seedCount-10]): "GET",
		fmt.Sprintf("http://localhost:8081/api/orders?ref=%s", refs[seedCount-5]):  "DELETE",
	}

	var r *http.Response
	var err error
	for url, verb := range urls {

		switch verb {
		case "GET":
			r, err = http.Get(url)

		case "DELETE":
			var req *http.Request
			req, err = http.NewRequest(http.MethodDelete, url, nil)
			if err != nil {
				t.Fatalf("Failed to create DELETE request: %v", err)
			}
			r, err = http.DefaultClient.Do(req)
		}

		if err != nil {
			t.Fatalf("Failed to make GET request: %v", err)
		}
		if r.StatusCode != http.StatusOK {
			t.Fatalf("Expected status OK, got %s", r.Status)
		}
	}

	err = server.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("Failed to shutdown server: %v", err)
	}

}
