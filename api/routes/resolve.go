package routes

import (
	"github.com/api/database"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
)

func ResolveURl(c *fiber.Ctx) error {

	url := c.Params("url")

	r := database.NewConnection(0) // will store original shorten urls and original urls
	defer r.Close()

	value, err := r.Get(database.Ctx, url).Result()
	if err == redis.Nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "redis cannot be found",
		})
	} else if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "cannot connect to DB",
		})
	}

	rInc := database.NewConnection(1) // this will give you the statistics and other analytics like the succesfull found of url
	defer rInc.Close()

	_ = rInc.Incr(database.Ctx, "counter")

	return c.Redirect(value, 301)

}
