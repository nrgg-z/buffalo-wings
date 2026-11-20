// Package httpx reúne piezas HTTP compartidas por todos los módulos:
// respuestas JSON con el formato estándar de la API, CORS e IP del cliente.
package httpx

import (
	"encoding/json"
	"log"
	"net/http"
)

// ErrorResponse es el formato único de error de la API:
// {"error": "codigo_corto", "message": "Texto para el usuario"}.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// WriteJSON responde con el cuerpo en JSON y el código indicado.
// Siempre agrega Cache-Control: no-store porque la API maneja datos sensibles
// (tokens, cuentas, datos de empleados) que no deben quedar en cachés.
func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("error escribiendo respuesta: %v", err)
	}
}

// WriteError responde con el formato estándar de error.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorResponse{Error: code, Message: message})
}
