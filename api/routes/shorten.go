package routes

import (
	"os"
	"strconv"
	"time"

	"github.com/api/database"
	"github.com/api/helpers"
	"github.com/asaskevich/govalidator"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type request struct {
	URL      string        `json:"url"`
	Shorturl string        `json:"short"`
	Expiry   time.Duration `json:"expiry"`
}

type response struct {
	URL            string        `json:"url"`
	Shorturl       string        `json:"short"`
	Expiry         time.Duration `json:"expiry"`
	XRateremaining int           `json:"rate_limit"`
	XRatelimitRest time.Duration `json:"rate_limit_remaining"`
}

func ShortenURL(c *fiber.Ctx) error {
	body := new(request)

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse the body",
		})
	}

	// implementing api limiter

	r2 := database.NewConnection(1)
	defer r2.Close()

	val, err := r2.Get(database.Ctx, c.IP()).Result()
	if err == redis.Nil {
		_ = r2.Set(database.Ctx, c.IP(), os.Getenv("API_QUOTA"), 30*60*time.Second).Err()
	} else {
		valInt, _ := strconv.Atoi(val)
		if valInt <= 0 {
			limit, _ := r2.TTL(database.Ctx, c.IP()).Result()
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":            "rate limit exceeded",
				"rate-limit-reset": limit / time.Nanosecond / time.Minute,
			})
		}
	}

	//check if the domain is correct or not

	if !govalidator.IsURL(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Wrong input",
		})
	}

	// now shorten the url

	if !helpers.RemoveDomainUrl(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "There is some error",
		})
	}

	body.URL = helpers.EnforceHTTP(body.URL)

	// fetch or check whether someone has used the same url or not
	var id string

	if body.Shorturl == "" {
		id = uuid.New().String()[:6]
	} else {
		id = body.Shorturl
	}

	r := database.NewConnection(0)

	val, _ = r.Get(database.Ctx, id).Result()
	if val != "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "The url is already in use",
		})
	}

	if body.Expiry == 0 {
		body.Expiry = 24
	}

	err = r2.Set(database.Ctx, id, body.URL, body.Expiry*3600*time.Second).Err()

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Unable to connect to server",
		})
	}

	resp := response{
		URL:            body.URL,
		Shorturl:       "",
		Expiry:         body.Expiry,
		XRateremaining: 10,
		XRatelimitRest: 30,
	}

	val, _ = r2.Get(database.Ctx, c.IP()).Result()
	resp.XRateremaining, _ = strconv.Atoi(val)

	ttl, _ := r2.TTL(database.Ctx, c.IP()).Result()
	resp.XRatelimitRest = ttl / time.Nanosecond / time.Minute

	resp.Shorturl = os.Getenv("DOMAIN") + "/" + id

	r2.Decr(database.Ctx, c.IP())

	return c.Status(fiber.StatusOK).JSON(resp)

}
