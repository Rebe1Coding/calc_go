package webrtc

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

const (
	SecretKey = "your-secret-key-change-in-production"
)

type LoginRequest struct {
	Username string `json:"username"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type SessionCreateRequest struct {
	TargetUsername string `json:"targetUsername"`
	Type           string `json:"type"`
}

type Session struct {
	SessionID string    `json:"sessionId"`
	Caller    string    `json:"caller"`
	Target    string    `json:"target"`
	Status    string    `json:"status"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
}

type WebSocketMessage struct {
	Event string                 `json:"event"`
	Data  map[string]interface{} `json:"data"`
}

type Claims struct {
	Username string `json:"sub"`
	jwt.RegisteredClaims
}

type Server struct {
	sessions      map[string]*Session
	userSessions  map[string]string
	wsConnections map[string]*websocket.Conn
	mu            sync.RWMutex
	upgrader      websocket.Upgrader
	port          string
}

func NewServer(port string) *Server {
	return &Server{
		sessions:      make(map[string]*Session),
		userSessions:  make(map[string]string),
		wsConnections: make(map[string]*websocket.Conn),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		port: port,
	}
}

func (s *Server) createToken(username string) (string, error) {
	claims := Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(60 * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(SecretKey))
}

func (s *Server) verifyToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims.Username, nil
	}

	return "", fmt.Errorf("invalid token")
}

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || len(authHeader) < 8 {
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		tokenString := authHeader[7:]
		username, err := s.verifyToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		r.Header.Set("X-Username", username)
		next(w, r)
	}
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}

	token, err := s.createToken(req.Username)
	if err != nil {
		http.Error(w, "Failed to create token", http.StatusInternalServerError)
		return
	}

	log.Printf("[WebRTC] User logged in: %s", req.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{Token: token})
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.Header.Get("X-Username")

	var req SessionCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.userSessions[username]; exists {
		http.Error(w, "Already in active session", http.StatusBadRequest)
		return
	}

	if _, exists := s.userSessions[req.TargetUsername]; exists {
		http.Error(w, "Target user is busy", http.StatusConflict)
		return
	}

	if req.TargetUsername == username {
		http.Error(w, "Cannot call yourself", http.StatusBadRequest)
		return
	}

	session := &Session{
		SessionID: uuid.New().String(),
		Caller:    username,
		Target:    req.TargetUsername,
		Status:    "pending",
		Type:      req.Type,
		CreatedAt: time.Now(),
	}

	s.sessions[session.SessionID] = session
	s.userSessions[username] = session.SessionID
	s.userSessions[req.TargetUsername] = session.SessionID

	log.Printf("[WebRTC] Session created: %s -> %s", username, req.TargetUsername)

	if conn, ok := s.wsConnections[req.TargetUsername]; ok {
		go func() {
			conn.WriteJSON(map[string]interface{}{
				"event": "session_updated",
				"data":  session,
			})
		}()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.Header.Get("X-Username")

	s.mu.RLock()
	defer s.mu.RUnlock()

	sessionID, exists := s.userSessions[username]
	if !exists {
		http.Error(w, "No active session", http.StatusNotFound)
		return
	}

	session, exists := s.sessions[sessionID]
	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (s *Server) handleAcceptSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.Header.Get("X-Username")

	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID, exists := s.userSessions[username]
	if !exists {
		http.Error(w, "No pending session", http.StatusNotFound)
		return
	}

	session, exists := s.sessions[sessionID]
	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	if session.Target != username {
		http.Error(w, "Only target can accept", http.StatusForbidden)
		return
	}

	if session.Status != "pending" {
		http.Error(w, "Session not pending", http.StatusBadRequest)
		return
	}

	session.Status = "active"

	log.Printf("[WebRTC] Session accepted: %s", sessionID)

	if conn, ok := s.wsConnections[session.Caller]; ok {
		go func() {
			conn.WriteJSON(map[string]interface{}{
				"event": "session_updated",
				"data":  session,
			})
		}()
	}

	if conn, ok := s.wsConnections[session.Target]; ok {
		go func() {
			conn.WriteJSON(map[string]interface{}{
				"event": "session_updated",
				"data":  session,
			})
		}()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (s *Server) handleDeclineSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.Header.Get("X-Username")

	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID, exists := s.userSessions[username]
	if !exists {
		http.Error(w, "No pending session", http.StatusNotFound)
		return
	}

	session, exists := s.sessions[sessionID]
	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	if session.Target != username {
		http.Error(w, "Only target can decline", http.StatusForbidden)
		return
	}

	session.Status = "declined"

	log.Printf("[WebRTC] Session declined: %s", sessionID)

	caller := session.Caller
	if conn, ok := s.wsConnections[caller]; ok {
		go func() {
			conn.WriteJSON(map[string]interface{}{
				"event": "session_updated",
				"data":  session,
			})
		}()
	}

	delete(s.userSessions, username)
	delete(s.userSessions, caller)
	delete(s.sessions, sessionID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (s *Server) handleCancelSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.Header.Get("X-Username")

	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID, exists := s.userSessions[username]
	if !exists {
		http.Error(w, "No active session", http.StatusNotFound)
		return
	}

	session, exists := s.sessions[sessionID]
	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	session.Status = "cancelled"

	log.Printf("[WebRTC] Session cancelled: %s", sessionID)

	otherUser := session.Target
	if session.Caller != username {
		otherUser = session.Caller
	}

	if conn, ok := s.wsConnections[otherUser]; ok {
		go func() {
			conn.WriteJSON(map[string]interface{}{
				"event": "session_updated",
				"data":  session,
			})
		}()
	}

	delete(s.userSessions, username)
	delete(s.userSessions, otherUser)
	delete(s.sessions, sessionID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Token required", http.StatusUnauthorized)
		return
	}

	username, err := s.verifyToken(token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebRTC] WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	s.mu.Lock()
	s.wsConnections[username] = conn
	s.mu.Unlock()

	log.Printf("[WebRTC] WebSocket connected: %s", username)

	for {
		var msg WebSocketMessage
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("[WebRTC] WebSocket read error for %s: %v", username, err)
			break
		}

		if msg.Event == "signal" {
			s.mu.RLock()
			sessionID, exists := s.userSessions[username]
			if !exists {
				s.mu.RUnlock()
				continue
			}

			session, exists := s.sessions[sessionID]
			if !exists || session.Status != "active" {
				s.mu.RUnlock()
				continue
			}

			otherUser := session.Target
			if session.Target == username {
				otherUser = session.Caller
			}

			otherConn, exists := s.wsConnections[otherUser]
			s.mu.RUnlock()

			if exists {
				go func() {
					otherConn.WriteJSON(map[string]interface{}{
						"event": "signal",
						"data":  msg.Data,
					})
				}()
				log.Printf("[WebRTC] Signal forwarded: %s -> %s", username, otherUser)
			}
		}
	}

	s.mu.Lock()
	delete(s.wsConnections, username)

	if sessionID, exists := s.userSessions[username]; exists {
		if session, exists := s.sessions[sessionID]; exists {
			session.Status = "disconnected"

			otherUser := session.Target
			if session.Target == username {
				otherUser = session.Caller
			}

			if otherConn, ok := s.wsConnections[otherUser]; ok {
				go func() {
					otherConn.WriteJSON(map[string]interface{}{
						"event": "session_updated",
						"data":  session,
					})
				}()
			}

			delete(s.userSessions, username)
			delete(s.userSessions, otherUser)
			delete(s.sessions, sessionID)
		}
	}
	s.mu.Unlock()

	log.Printf("[WebRTC] WebSocket disconnected: %s", username)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"name":    "WebRTC Signaling Server",
		"version": "1.0.0",
		"endpoints": map[string]string{
			"auth":      "/api/auth/login",
			"session":   "/api/session",
			"websocket": "/ws?token=YOUR_JWT_TOKEN",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleSessionRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch path {
	case "/api/session":
		switch r.Method {
		case http.MethodPost:
			s.authMiddleware(s.handleCreateSession)(w, r)
		case http.MethodGet:
			s.authMiddleware(s.handleGetSession)(w, r)
		case http.MethodDelete:
			s.authMiddleware(s.handleCancelSession)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "/api/session/accept":
		s.authMiddleware(s.handleAcceptSession)(w, r)
	case "/api/session/decline":
		s.authMiddleware(s.handleDeclineSession)(w, r)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// InitiateCall - метод для инициации звонка программно (для интерпретатора)
func (s *Server) InitiateCall(caller, target, callType string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Проверки
	if _, exists := s.userSessions[caller]; exists {
		return nil, fmt.Errorf("caller already in active session")
	}

	if _, exists := s.userSessions[target]; exists {
		return nil, fmt.Errorf("target user is busy")
	}

	if target == caller {
		return nil, fmt.Errorf("cannot call yourself")
	}

	// Создаём сессию
	session := &Session{
		SessionID: uuid.New().String(),
		Caller:    caller,
		Target:    target,
		Status:    "pending",
		Type:      callType,
		CreatedAt: time.Now(),
	}

	s.sessions[session.SessionID] = session
	s.userSessions[caller] = session.SessionID
	s.userSessions[target] = session.SessionID

	log.Printf("[WebRTC] Session created programmatically: %s -> %s", caller, target)

	// Уведомляем получателя через WebSocket
	if conn, ok := s.wsConnections[target]; ok {
		go func() {
			conn.WriteJSON(map[string]interface{}{
				"event": "session_updated",
				"data":  session,
			})
		}()
	}

	return session, nil
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/api/auth/login", s.handleLogin)
	mux.HandleFunc("/api/session", s.handleSessionRoutes)
	mux.HandleFunc("/api/session/accept", s.handleSessionRoutes)
	mux.HandleFunc("/api/session/decline", s.handleSessionRoutes)
	mux.HandleFunc("/ws", s.handleWebSocket)

	// Serve static files
	mux.Handle("/webrtc/", http.StripPrefix("/webrtc/", http.FileServer(http.Dir("./static/webrtc"))))

	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}).Handler(mux)

	log.Printf("🚀 WebRTC Signaling Server starting on %s", s.port)
	log.Printf("📡 WebSocket endpoint: ws://localhost%s/ws", s.port)
	return http.ListenAndServe(s.port, handler)
}
