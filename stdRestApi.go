package elorm

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
)

func sendHttpError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(html.EscapeString(message)))
}

// RestApiConfig is a configuration for standard REST API operations.
type RestApiConfig[T IEntity] struct {
	Def                EntityDef
	LoadEntityFunc     func(ref string) (*T, error)
	SelectEntitiesFunc func(filters []*Filter, sorts []*SortItem, pageNo int, pageSize int) (result []*T, pagesCount int, err error)
	CreateEntityFunc   func() (*T, error)
	AdditionalHeaders  map[string]string
	AutoFilters        bool
	DefaultPageSize    int

	EnableGetOne     bool
	EnableGetList    bool
	EnablePost       bool
	EnablePut        bool
	EnableDelete     bool
	EnableSoftDelete bool

	ParamRef      string
	ParamPageNo   string
	ParamPageSize string
	ParamSortBy   string

	BeforeMiddleware func(http.ResponseWriter, *http.Request) bool
	Context          func(r *http.Request) context.Context
	AdditionalFilter func(r *http.Request) ([]*Filter, error)   //for list
	DefaultSorts     func(r *http.Request) ([]*SortItem, error) //for list
}

// DefaultPageSize is the default number of items per page in REST API responses.
const DefaultPageSize = 20

// CreateStdRestApiConfig creates a new RestApiConfig for standard REST API operations.
func CreateStdRestApiConfig[T IEntity](
	def EntityDef,
	loadEntityFunc func(ref string) (*T, error),
	selectEntitiesFunc func(filters []*Filter, sorts []*SortItem, pageNo int, pageSize int) (result []*T, pages int, err error),
	createEntityFunc func() (*T, error)) RestApiConfig[T] {
	return RestApiConfig[T]{
		Def:                def,
		LoadEntityFunc:     loadEntityFunc,
		SelectEntitiesFunc: selectEntitiesFunc,
		CreateEntityFunc:   createEntityFunc,
		AdditionalHeaders:  make(map[string]string),
		AutoFilters:        true,
		DefaultPageSize:    DefaultPageSize,

		EnableGetOne:     true,
		EnableGetList:    true,
		EnablePost:       true,
		EnablePut:        true,
		EnableDelete:     true,
		EnableSoftDelete: true,

		ParamRef:      "ref",
		ParamPageNo:   "pageno",
		ParamPageSize: "pagesize",
		ParamSortBy:   "sortby",
	}
}

// HandleRestApi handles HTTP requests for the REST API based on the provided configuration.
func HandleRestApi[T IEntity](config RestApiConfig[T]) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {

		if config.BeforeMiddleware != nil {
			if !config.BeforeMiddleware(w, r) {
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if len(config.AdditionalHeaders) > 0 {
			for key, value := range config.AdditionalHeaders {
				w.Header().Set(key, value)
			}
		}
		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Has(config.ParamRef) {
				if config.EnableGetOne {
					responseGet(config, r, w)
				} else {
					sendHttpError(w, "", http.StatusNotFound)
				}
				return
			} else {
				if config.EnableGetList {
					responseGetList(config, r, w)
				} else {
					sendHttpError(w, "", http.StatusNotFound)
				}
				return
			}
		case http.MethodPost:
			if config.EnablePost {
				responsePost(config, w, r)
			} else {
				sendHttpError(w, "", http.StatusNotFound)
			}
			return
		case http.MethodPut:
			if config.EnablePut {
				if r.URL.Query().Has(config.ParamRef) {
					responsePut(config, r, w)
				} else {
					sendHttpError(w, fmt.Sprintf("Missing '%s' parameter", config.ParamRef), http.StatusBadRequest)
				}
			} else {
				sendHttpError(w, "", http.StatusNotFound)
			}
		case http.MethodDelete:
			if config.EnableDelete {
				responseDelete(config, w, r)
			} else {
				sendHttpError(w, "", http.StatusNotFound)
			}
		default:
			sendHttpError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func responsePut[T IEntity](config RestApiConfig[T], r *http.Request, w http.ResponseWriter) {
	const methodPrefix = "RestApiConfig.responsePut: "
	ref := r.URL.Query().Get(config.ParamRef)
	if ref == "" {
		sendHttpError(w, fmt.Sprintf("%smissing ref parameter", methodPrefix), http.StatusBadRequest)
		return
	}

	dbRecord, err := config.LoadEntityFunc(ref)
	if err != nil {
		sendHttpError(w, fmt.Sprintf("%sfailed to load entity: %v", methodPrefix, err), http.StatusInternalServerError)
		return
	}

	reqRecord, err := config.CreateEntityFunc()
	if err != nil {
		sendHttpError(w, fmt.Sprintf("%sfailed to create entity: %v", methodPrefix, err), http.StatusInternalServerError)
		return
	}

	err = json.NewDecoder(r.Body).Decode(&reqRecord)
	if err != nil {
		sendHttpError(w, fmt.Sprintf("%sinvalid request data: %v", methodPrefix, err), http.StatusBadRequest)
		return
	}

	err = (*dbRecord).LoadFrom((*reqRecord), false)
	if err != nil {
		sendHttpError(w, fmt.Sprintf("%sfailed to load data into entity: %v", methodPrefix, err), http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	if config.Context != nil {
		ctx = config.Context(r)
	}
	err = (*dbRecord).Save(ctx)
	if err != nil {
		sendHttpError(w, fmt.Sprintf("%sfailed to save entity: %v", methodPrefix, err), http.StatusInternalServerError)
		return
	}
}

func responsePost[T IEntity](config RestApiConfig[T], w http.ResponseWriter, r *http.Request) {
	const methodPrefix = "RestApiConfig.responsePost: "
	newRecord, err := config.CreateEntityFunc()
	refFld := (*newRecord).GetValues()[RefFieldName].(*FieldValueRef)
	oldRef := refFld.v
	if err != nil {
		sendHttpError(w, fmt.Sprintf("%sfailed to create entity: %v", methodPrefix, err), http.StatusInternalServerError)
		return
	}

	err = json.NewDecoder(r.Body).Decode(&newRecord)
	if err != nil {
		sendHttpError(w, fmt.Sprintf("%sinvalid request data: %v", methodPrefix, err), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if config.Context != nil {
		ctx = config.Context(r)
	}

	// override generated Ref after load from request body
	if refFld.v == "" {
		refFld.v = oldRef
	}
	err = (*newRecord).Save(ctx)
	if err != nil {
		sendHttpError(w, fmt.Sprintf("%sfailed to save entity: %v", methodPrefix, err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(newRecord)
	if err != nil {
		sendHttpError(w, fmt.Sprintf("%sfailed to encode entity to JSON: %v", methodPrefix, err), http.StatusInternalServerError)
		return
	}
}

func responseDelete[T IEntity](config RestApiConfig[T], w http.ResponseWriter, r *http.Request) {
	const methodPrefix = "RestApiConfig.responseDelete: "
	if r.URL.Query().Has(config.ParamRef) {
		ref := r.URL.Query().Get(config.ParamRef)

		ctx := r.Context()
		if config.Context != nil {
			ctx = config.Context(r)
		}

		if config.EnableSoftDelete {
			ent, err := config.Def.Factory.LoadEntity(ref)
			if err != nil {
				sendHttpError(w, fmt.Sprintf("%sfailed to load entity: %v", methodPrefix, err), http.StatusNotFound)
				return
			}
			if !ent.IsDeleted() {
				ent.SetIsDeleted(true)
				err = ent.Save(ctx)
				if err != nil {
					sendHttpError(w, fmt.Sprintf("%sfailed to soft delete entity: %v", methodPrefix, err), http.StatusInternalServerError)
					return
				}
			}
		} else {
			err := config.Def.Factory.DeleteEntity(ctx, ref)
			if err != nil {
				sendHttpError(w, fmt.Sprintf("%sfailed to delete entity: %v", methodPrefix, err), http.StatusNotFound)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
	} else {
		sendHttpError(w, fmt.Sprintf("%smissing ref parameter", methodPrefix), http.StatusNotFound)
	}
}

func responseGet[T IEntity](config RestApiConfig[T], r *http.Request, w http.ResponseWriter) {
	const methodPrefix = "RestApiConfig.responseGet: "
	ref := r.URL.Query().Get(config.ParamRef)
	if ref == "" {
		sendHttpError(w, fmt.Sprintf("%smissing ref parameter", methodPrefix), http.StatusBadRequest)
		return
	}

	record, err := config.LoadEntityFunc(ref)
	if err != nil {
		sendHttpError(w, fmt.Sprintf("%sfailed to load entity: %v", methodPrefix, err), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(record)
	if err != nil {
		sendHttpError(w, fmt.Sprintf("%sfailed to encode entity to JSON: %v", methodPrefix, err), http.StatusInternalServerError)
		return
	}
}

func responseGetList[T IEntity](config RestApiConfig[T], r *http.Request, w http.ResponseWriter) {
	const methodPrefix = "RestApiConfig.responseGetList: "
	var pageNo int
	var err error
	pageNoStr := r.URL.Query().Get(config.ParamPageNo)
	if pageNo, err = strconv.Atoi(pageNoStr); err != nil {
		pageNo = 1
	}

	var pageSize int
	pageSizeStr := r.URL.Query().Get(config.ParamPageSize)
	if pageSize, err = strconv.Atoi(pageSizeStr); err != nil {
		pageSize = config.DefaultPageSize
	}

	filters := make([]*Filter, 0)
	if config.AutoFilters {
		for k, v := range r.URL.Query() {
			if k == config.ParamRef || k == config.ParamPageNo || k == config.ParamPageSize {
				continue
			}
			fd := config.Def.FieldDefByName(k)
			if fd != nil {
				filters = append(filters, AddFilterEQ(fd, v[0]))
			}
		}
	}

	foundsSorts := make([]*SortItem, 0)

	sorts := make([]*SortItem, 0)

	if config.DefaultSorts != nil {
		sorts, err = config.DefaultSorts(r)
		if err != nil {
			sendHttpError(w, fmt.Sprintf("%sfailed to get default sorts: %v", methodPrefix, err), http.StatusInternalServerError)
			return
		}
	} else {
		sorts = append(sorts, &SortItem{Field: config.Def.RefField, Asc: true})
	}

	if sortBy := r.URL.Query().Get(config.ParamSortBy); sortBy != "" {
		sortParts := strings.Split(sortBy, ",")
		for _, token := range sortParts {
			token = strings.ToLower(strings.TrimSpace(token))
			if token == "" {
				continue
			}
			parts := strings.Split(token, " ")
			if len(parts) == 2 {
				def := config.Def.FieldDefByName(parts[0])
				if def != nil {
					asc := true
					if strings.ToLower(parts[1]) == "desc" {
						asc = false
					}
					foundsSorts = append(foundsSorts, &SortItem{Field: def, Asc: asc})
				}
			}
		}
		if len(foundsSorts) > 0 {
			sorts = foundsSorts
		}
	}

	if config.AdditionalFilter != nil {
		aFilters, err := config.AdditionalFilter(r)
		if err != nil {
			sendHttpError(w, fmt.Sprintf("%sfailed to get additional filters: %v", methodPrefix, err), http.StatusInternalServerError)
			return
		}
		for _, f := range aFilters {
			if f != nil {

				found := false
				for _, existFilter := range filters {
					if existFilter.LeftOp == f.LeftOp {
						existFilter.Op = f.Op
						existFilter.RightOp = f.RightOp
						existFilter.Childs = f.Childs
						found = true
						break
					}
				}
				if !found {
					filters = append(filters, f)
				}
			}
		}
	}

	records, pagesCount, err := config.SelectEntitiesFunc(filters, sorts, pageNo, pageSize)
	if err != nil {
		sendHttpError(w, fmt.Sprintf("%sfailed to fetch list: %v", methodPrefix, err), http.StatusInternalServerError)
		return
	}

	response := struct {
		Data       []*T
		PagesCount int
	}{
		Data:       records,
		PagesCount: pagesCount,
	}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		sendHttpError(w, fmt.Sprintf("%sfailed to encode list to JSON: %v", methodPrefix, err), http.StatusInternalServerError)
		return
	}
}
