package server

import (
	"github.com/gofiber/fiber/v2"
)

func (s *Server) parseAndValidate(c *fiber.Ctx, reqP any) error {

	err := c.BodyParser(reqP)
	if err != nil {
		return err
	}
	err = s.validator.Struct(reqP)
	if err != nil {
		return err
	}
	return nil
}
