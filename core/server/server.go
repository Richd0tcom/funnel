package server

import "github.com/gin-gonic/gin"

type Server struct {
	Router *gin.Engine
}

func NewServer() (*Server, error) {
	return &Server{}, nil
}