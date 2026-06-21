package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/blocstor/bloc-agent/internal/exec"
)

// Server holds the HTTP mux and logger for the agent API.
type Server struct {
	mux      *http.ServeMux
	logger   *slog.Logger
	thinpool string
}

// NewServer creates a new Server, registers all routes, and returns it.
// thinpool is optional: when non-empty, LV creation uses thin provisioning from this pool.
func NewServer(logger *slog.Logger, thinpool string) *Server {
	s := &Server{
		mux:      http.NewServeMux(),
		logger:   logger,
		thinpool: thinpool,
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

	s.mux.HandleFunc("POST /drbd/create-md", s.handleDRBDCreateMD)
	s.mux.HandleFunc("POST /drbd/up", s.handleDRBDUp)
	s.mux.HandleFunc("POST /drbd/down", s.handleDRBDDown)
	s.mux.HandleFunc("POST /drbd/primary", s.handleDRBDPrimary)
	s.mux.HandleFunc("POST /drbd/primary-force", s.handleDRBDPrimaryForce)
	s.mux.HandleFunc("POST /drbd/secondary", s.handleDRBDSecondary)
	s.mux.HandleFunc("POST /drbd/resize", s.handleDRBDResize)
	s.mux.HandleFunc("GET /drbd/status", s.handleDRBDStatus)

	s.mux.HandleFunc("GET /vm/blklist", s.handleVMBlklist)
	s.mux.HandleFunc("POST /vm/attach", s.handleVMAttach)
	s.mux.HandleFunc("POST /vm/detach", s.handleVMDetach)

	s.mux.HandleFunc("POST /nbd/serve", s.handleNBDServe)
	s.mux.HandleFunc("POST /nbd/stop", s.handleNBDStop)

	s.mux.HandleFunc("GET /quorum", s.handleQuorum)
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
	s.logger.Info("lv create", "vg", req.VG, "name", req.Name, "size_mb", req.SizeMB, "thinpool", s.thinpool)
	if err := exec.CreateLV(req.VG, req.Name, s.thinpool, req.SizeMB); err != nil {
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

func (s *Server) handleDRBDCreateMD(w http.ResponseWriter, r *http.Request) {
	var req drbdResourceRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.Resource == "" {
		writeError(w, http.StatusBadRequest, "resource is required")
		return
	}
	s.logger.Info("drbd create-md", "resource", req.Resource)
	if err := exec.CreateMD(req.Resource); err != nil {
		s.logger.Error("drbd create-md failed", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

func (s *Server) handleDRBDPrimaryForce(w http.ResponseWriter, r *http.Request) {
	var req drbdResourceRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.Resource == "" {
		writeError(w, http.StatusBadRequest, "resource is required")
		return
	}
	s.logger.Info("drbd primary --force", "resource", req.Resource)
	if err := exec.PrimaryForce(req.Resource); err != nil {
		s.logger.Error("drbd primary --force failed", "err", err)
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

// VM handlers

func (s *Server) handleVMBlklist(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	if domain == "" {
		writeError(w, http.StatusBadRequest, "domain query parameter is required")
		return
	}
	s.logger.Info("vm blklist", "domain", domain)
	targets, err := exec.VMBlockList(domain)
	if err != nil {
		s.logger.Error("vm blklist failed", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string][]string{"targets": targets})
}

type vmAttachRequest struct {
	Domain string `json:"domain"`
	Source string `json:"source"`
	Target string `json:"target"`
}

func (s *Server) handleVMAttach(w http.ResponseWriter, r *http.Request) {
	var req vmAttachRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.Domain == "" || req.Source == "" || req.Target == "" {
		writeError(w, http.StatusBadRequest, "domain, source, and target are required")
		return
	}
	s.logger.Info("vm attach", "domain", req.Domain, "source", req.Source, "target", req.Target)
	if err := exec.VMAttach(req.Domain, req.Source, req.Target); err != nil {
		s.logger.Error("vm attach failed", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type vmDetachRequest struct {
	Domain string `json:"domain"`
	Target string `json:"target"`
}

func (s *Server) handleVMDetach(w http.ResponseWriter, r *http.Request) {
	var req vmDetachRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.Domain == "" || req.Target == "" {
		writeError(w, http.StatusBadRequest, "domain and target are required")
		return
	}
	s.logger.Info("vm detach", "domain", req.Domain, "target", req.Target)
	if err := exec.VMDetach(req.Domain, req.Target); err != nil {
		s.logger.Error("vm detach failed", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// NBD handlers

type nbdServeRequest struct {
	Device string `json:"device"`
	Port   int    `json:"port"`
}

func (s *Server) handleNBDServe(w http.ResponseWriter, r *http.Request) {
	var req nbdServeRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.Device == "" || req.Port <= 0 {
		writeError(w, http.StatusBadRequest, "device and port are required")
		return
	}
	s.logger.Info("nbd serve", "device", req.Device, "port", req.Port)
	if err := exec.NBDServe(req.Device, req.Port); err != nil {
		s.logger.Error("nbd serve failed", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type nbdStopRequest struct {
	Port int `json:"port"`
}

func (s *Server) handleNBDStop(w http.ResponseWriter, r *http.Request) {
	var req nbdStopRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.Port <= 0 {
		writeError(w, http.StatusBadRequest, "port is required")
		return
	}
	s.logger.Info("nbd stop", "port", req.Port)
	if err := exec.NBDStop(req.Port); err != nil {
		s.logger.Error("nbd stop failed", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleQuorum(w http.ResponseWriter, r *http.Request) {
	q, err := exec.ClusterQuorate()
	if err != nil {
		s.logger.Error("quorum check failed", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"quorate": q})
}
