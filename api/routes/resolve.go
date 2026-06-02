package routes

import (
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/rohilprajapati/shortern_url_redis_fiber/database"
)

func ResolveURL(c *fiber.Ctx) error {
	url := c.Params("url")

	r := database.CreateClient(0)

	defer r.Close()

	value, err := r.Get(database.Ctx, url).Result()

	if err == redis.Nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Short not found"})
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Something went wrong, please try again later"})
	}
	rInr := database.CreateClient(1)
	defer rInr.Close()

	rInr.Incr(database.Ctx, url+"_clicks")

	return c.Redirect(value, 301)

}
