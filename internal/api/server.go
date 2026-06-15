package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/blocstor/bloc-agent/internal/exec"
)

// Server holds the HTTP mux and logger for the agent API.
type Server struct {
	mux    *http.ServeMux
	logger *slog.Logger
}

// NewServer creates a new Server, registers all routes, and returns it.
func NewServer(logger *slog.Logger) *Server {
	s := &Server{
		mux:    http.NewServeMux(),
		logger: logger,
	}
	s.registerRoutes()
	return s
}

// ServeHTTP implements http.Handler so *Server can be used directly.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)

	s.mux.HandleFunc("POST /lv/create", s.handleLVCreate)
	s.mux.HandleFunc("POST /lv/extend", s.handleLVExtend)
	s.mux.HandleFunc("POST /lv/remove", s.handleLVRemove)

	s.mux.HandleFunc("POST /res/write", s.handleResWrite)
	s.mux.HandleFunc("POST /res/remove", s.handleResRemove)

	s.mux.HandleFunc("POST /drbd/up", s.handleDRBDUp)
	s.mux.HandleFunc("POST /drbd/down", s.handleDRBDDown)
	s.mux.HandleFunc("POST /drbd/primary", s.handleDRBDPrimary)
	s.mux.HandleFunc("POST /drbd/secondary", s.handleDRBDSecondary)
	s.mux.HandleFunc("POST /drbd/resize", s.handleDRBDResize)
	s.mux.HandleFunc("GET /drbd/status", s.handleDRBDStatus)
}

// writeError writes a JSON error response with the given HTTP status code.
func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// writeJSON writes a JSON response with status 200.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// decodeBody decodes the JSON request body into dst.
func decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

// handleHealthz responds 200 OK to liveness probes.
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// LV handlers

type lvCreateRequest struct {
	Name   string `json:"name"`
	VG     string `json:"vg"`
	SizeMB int    `json:"size_mb"`
}

func (s *Server) handleLVCreate(w http.ResponseWriter, r *http.Request) {
	var req lvCreateRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.Name == "" || req.VG == "" || req.SizeMB <= 0 {
		writeError(w, http.StatusBadRequest, "name, vg, and size_mb (>0) are required")
		return
	}
	s.logger.Info("lv create", "vg", req.VG, "name", req.Name, "size_mb", req.SizeMB)
	if err := exec.CreateLV(req.VG, req.Name, req.SizeMB); err != nil {
		s.logger.Error("lv create failed", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type lvExtendRequest struct {
	Name  string `json:"name"`
	VG    string `json:"vg"`
	AddMB int    `json:"add_mb"`
}

func (s *Server) handleLVExtend(w http.ResponseWriter, r *http.Request) {
	var req lvExtendRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.Name == "" || req.VG == "" || req.AddMB <= 0 {
		writeError(w, http.StatusBadRequest, "name, vg, and add_mb (>0) are required")
		return
	}
	s.logger.Info("lv extend", "vg", req.VG, "name", req.Name, "add_mb", req.AddMB)
	if err := exec.ExtendLV(req.VG, req.Name, req.AddMB); err != nil {
		s.logger.Error("lv extend failed", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type lvRemoveRequest struct {
	Name string `json:"name"`
	VG   string `json:"vg"`
}

func (s *Server) handleLVRemove(w http.ResponseWriter, r *http.Request) {
	var req lvRemoveRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.Name == "" || req.VG == "" {
		writeError(w, http.StatusBadRequest, "name and vg are required")
		return
	}
	s.logger.Info("lv remove", "vg", req.VG, "name", req.Name)
	if err := exec.RemoveLV(req.VG, req.Name); err != nil {
		s.logger.Error("lv remove failed", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Resource file handlers

type resWriteRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func (s *Server) handleResWrite(w http.ResponseWriter, r *http.Request) {
	var req resWriteRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	s.logger.Info("res write", "name", req.Name)
	if err := exec.WriteRes(req.Name, req.Content); err != nil {
		s.logger.Error("res write failed", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type resRemoveRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleResRemove(w http.ResponseWriter, r *http.Request) {
	var req resRemoveRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	s.logger.Info("res remove", "name", req.Name)
	if err := exec.RemoveRes(req.Name); err != nil {
		s.logger.Error("res remove failed", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DRBD handlers

type drbdResourceRequest struct {
	Resource string `json:"resource"`
}

func (s *Server) handleDRBDUp(w http.ResponseWriter, r *http.Request) {
	var req drbdResourceRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.Resource == "" {
		writeError(w, http.StatusBadRequest, "resource is required")
		return
	}
	s.logger.Info("drbd up", "resource", req.Resource)
	if err := exec.Up(req.Resource); err != nil {
		s.logger.Error("drbd up failed", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDRBDDown(w http.ResponseWriter, r *http.Request) {
	var req drbdResourceRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.Resource == "" {
		writeError(w, http.StatusBadRequest, "resource is required")
		return
	}
	s.logger.Info("drbd down", "resource", req.Resource)
	if err := exec.Down(req.Resource); err != nil {
		s.logger.Error("drbd down failed", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDRBDPrimary(w http.ResponseWriter, r *http.Request) {
	var req drbdResourceRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.Resource == "" {
		writeError(w, http.StatusBadRequest, "resource is required")
		return
	}
	s.logger.Info("drbd primary", "resource", req.Resource)
	if err := exec.Primary(req.Resource); err != nil {
		s.logger.Error("drbd primary failed", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDRBDSecondary(w http.ResponseWriter, r *http.Request) {
	var req drbdResourceRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.Resource == "" {
		writeError(w, http.StatusBadRequest, "resource is required")
		return
	}
	s.logger.Info("drbd secondary", "resource", req.Resource)
	if err := exec.Secondary(req.Resource); err != nil {
		s.logger.Error("drbd secondary failed", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDRBDResize(w http.ResponseWriter, r *http.Request) {
	var req drbdResourceRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.Resource == "" {
		writeError(w, http.StatusBadRequest, "resource is required")
		return
	}
	s.logger.Info("drbd resize", "resource", req.Resource)
	if err := exec.Resize(req.Resource); err != nil {
		s.logger.Error("drbd resize failed", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDRBDStatus(w http.ResponseWriter, r *http.Request) {
	resource := r.URL.Query().Get("resource")
	if resource == "" {
		writeError(w, http.StatusBadRequest, "resource query parameter is required")
		return
	}
	s.logger.Info("drbd status", "resource", resource)
	out, err := exec.Status(resource)
	if err != nil {
		s.logger.Error("drbd status failed", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"output": out})
}
