package server

import "github.com/gofiber/fiber/v2"

func (s *Server) liveness(c *fiber.Ctx) error {
	_, span := s.tracer.Start(c.UserContext(), "server.liveness")
	defer span.End()
	return c.SendString("OK")
}
