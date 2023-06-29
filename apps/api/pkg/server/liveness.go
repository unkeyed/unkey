package server

import "github.com/gofiber/fiber/v2"

func (s *Server) liveness(c *fiber.Ctx) error {
	return c.SendString("OK")
}
