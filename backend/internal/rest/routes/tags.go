package routes

import (
	"fmt"
	"net/http"
)

func (h *Handlers) registerTags(mux *http.ServeMux) {
	// The tags in memory, with their last values
	HandleFunc(mux, "GET /getTags", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		SendJSON(response, http.StatusOK, h.Tags.Tags())
	}))

	// The rows of the tag table
	HandleFunc(mux, "GET /tags", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		tags, err := h.Tags.ReadTagsFromDB(request.Context())
		if err != nil {
			SendError(response, err, "", "Error fetching tags.")
			return
		}
		SendJSON(response, http.StatusOK, tags)
	}))

	HandleFunc(mux, "POST /tags", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		var body TagCreateSchema
		if err := ParseBody(request, &body); err != nil {
			SendMessage(response, http.StatusBadRequest, "Invalid request body: "+err.Error())
			return
		}
		if message := body.Validate(); message != "" {
			SendMessage(response, http.StatusBadRequest, message)
			return
		}

		tag, err := h.Tags.CreateTagInDB(request.Context(), body.Tag.ToTag())
		if err != nil {
			SendError(response, err, "", "Error creating tag.")
			return
		}
		SendJSON(response, http.StatusCreated, tag)
	}))

	HandleFunc(mux, "PUT /tags/{id}", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		id, ok := ParseIDParam(response, request, "id")
		if !ok {
			return
		}
		var body TagUpdateSchema
		if err := ParseBody(request, &body); err != nil {
			SendMessage(response, http.StatusBadRequest, "Invalid request body: "+err.Error())
			return
		}
		if message := body.Validate(); message != "" {
			SendMessage(response, http.StatusBadRequest, message)
			return
		}

		tag, err := h.Tags.UpdateTagInDB(request.Context(), id, body.Tag.ToTag())
		if err != nil {
			SendError(response, err, fmt.Sprintf("Tag with ID %d not found.", id), "Error updating tag.")
			return
		}
		SendJSON(response, http.StatusOK, tag)
	}))
}
