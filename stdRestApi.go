package elorm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

func SendHttpError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	w.Write([]byte(message))
}

type RestApiConfig[T IEntity] struct {
	Def                EntityDef
	LoadEntityFunc     func(ref string) (*T, error)
	SelectEntitiesFunc func(filters []*Filter, sorts []*SortItem, pageNo int, pageSize int) (result []*T, pages int, err error)
	CreateEntityFunc   func() (*T, error)
	AdditionalHeaders  map[string]string
	AutoFilters        bool
	DefaultPageSize    int

	EnableGetOne  bool
	EnableGetList bool
	EnablePost    bool
	EnablePut     bool
	EnableDelete  bool

	ParamRef      string
	ParamPageNo   string
	ParamPageSize string
}

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
		DefaultPageSize:    20,

		EnableGetOne:  true,
		EnableGetList: true,
		EnablePost:    true,
		EnablePut:     true,
		EnableDelete:  true,

		ParamRef:      "ref",
		ParamPageNo:   "pageno",
		ParamPageSize: "pagesize",
	}
}

func HandleRestApi[T IEntity](config RestApiConfig[T]) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
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
					SendHttpError(w, "", http.StatusNotFound)
				}
				return
			} else {
				if config.EnableGetList {
					responseGetList(config, r, w)
				} else {
					SendHttpError(w, "", http.StatusNotFound)
				}
				return
			}
		case http.MethodPost:
			if config.EnablePost {
				responsePost(config, w, r)
			} else {
				SendHttpError(w, "", http.StatusNotFound)
			}
			return
		case http.MethodPut:
			if config.EnablePut {
				if r.URL.Query().Has(config.ParamRef) {
					responsePut(config, r, w)
				} else {
					SendHttpError(w, fmt.Sprintf("Missing '%s' parameter", config.ParamRef), http.StatusBadRequest)
				}
			} else {
				SendHttpError(w, "", http.StatusNotFound)
			}
		case http.MethodDelete:
			if config.EnableDelete {
				responseDelete(config, w, r)
			} else {
				SendHttpError(w, "", http.StatusNotFound)
			}
		default:
			SendHttpError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func responsePut[T IEntity](config RestApiConfig[T], r *http.Request, w http.ResponseWriter) {
	ref := r.URL.Query().Get(config.ParamRef)

	dbRecord, err := config.LoadEntityFunc(ref)
	if err != nil {
		SendHttpError(w, fmt.Sprintf("Error fetching record: %v", err), http.StatusInternalServerError)
		return
	}

	reqRecord, err := config.CreateEntityFunc()
	if err != nil {
		SendHttpError(w, fmt.Sprintf("Error creating record: %v", err), http.StatusInternalServerError)
		return
	}
	err = json.NewDecoder(r.Body).Decode(&reqRecord)
	if err != nil {
		SendHttpError(w, "Invalid request data", http.StatusBadRequest)
		return
	}

	(*dbRecord).LoadFrom((*reqRecord), false)

	err = (*dbRecord).Save()
	if err != nil {
		SendHttpError(w, fmt.Sprintf("Error saving user: %v", err), http.StatusInternalServerError)
		return
	}
}

func responsePost[T IEntity](config RestApiConfig[T], w http.ResponseWriter, r *http.Request) {
	newRecord, err := config.CreateEntityFunc()
	if err != nil {
		SendHttpError(w, fmt.Sprintf("Error creating record: %v", err), http.StatusInternalServerError)
		return
	}
	err = json.NewDecoder(r.Body).Decode(&newRecord)
	if err != nil {
		SendHttpError(w, "Invalid request data", http.StatusBadRequest)
		return
	}
	err = (*newRecord).Save()
	if err != nil {
		SendHttpError(w, fmt.Sprintf("Error saving user: %v", err), http.StatusInternalServerError)
		return
	}
}

func responseDelete[T IEntity](config RestApiConfig[T], w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Has(config.ParamRef) {
		ref := r.URL.Query().Get(config.ParamRef)
		err := config.Def.Factory.DeleteEntity(ref)
		if err != nil {
			SendHttpError(w, fmt.Sprintf("Error deleting record: %w", err), http.StatusNotFound)
		}
		w.WriteHeader(http.StatusOK)
	} else {
		SendHttpError(w, "", http.StatusNotFound)
	}
}

func responseGet[T IEntity](config RestApiConfig[T], r *http.Request, w http.ResponseWriter) {
	ref := r.URL.Query().Get(config.ParamRef)

	record, err := config.LoadEntityFunc(ref)
	if err != nil {
		SendHttpError(w, fmt.Sprintf("Error fetching record: %v", err), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(record)
	if err != nil {
		SendHttpError(w, fmt.Sprintf("Converting record error to JSON: %v", err), http.StatusInternalServerError)
		return
	}
}

func responseGetList[T IEntity](config RestApiConfig[T], r *http.Request, w http.ResponseWriter) {
	pageNo := 1
	var err error
	pageNoStr := r.URL.Query().Get(config.ParamPageNo)
	if pageNo, err = strconv.Atoi(pageNoStr); err != nil {
		pageNo = 1
	}

	var pageSize int = config.DefaultPageSize
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

	sorts := make([]*SortItem, 0)
	sorts = append(sorts, &SortItem{Field: config.Def.RefField, Asc: true})

	records, total, err := config.SelectEntitiesFunc(filters, sorts, pageNo, pageSize)
	if err != nil {
		SendHttpError(w, fmt.Sprintf("Error fetching list: %v", err), http.StatusInternalServerError)
		return
	}

	response := struct {
		Data  []*T
		Total int
	}{
		Data:  records,
		Total: total,
	}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		SendHttpError(w, fmt.Sprintf("Error converting list to JSON: %v", err), http.StatusInternalServerError)
		return
	}
}
