package routes

import (
	"os"
	"strconv"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rohilprajapati/shortern_url_redis_fiber/database"
	"github.com/rohilprajapati/shortern_url_redis_fiber/helpers"
)

type request struct {
	URL         string        `json:"url"`
	CustomShort string        `json:"short"`
	Expiry      time.Duration `json:"expiry"`
}

type response struct {
	URL             string        `json:"url"`
	CustomShort     string        `json:"short"`
	Expiry          time.Duration `json:"expiry"`
	XRateRemaining  int           `json:"rate_limit"`
	XRateLimitReset time.Duration `json:"rate_limit_reset"`
}

func ShortenURL(c *fiber.Ctx) error {
	body := new(request)

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	// implement rate limiting
	dbRateStr := os.Getenv("REDIS_DB_RATE")

	dbRate, err := strconv.Atoi(dbRateStr)
	if err != nil {
		dbRate = 1
	}

	r2 := database.CreateClient(dbRate)
	defer r2.Close()

	val, err := r2.Get(database.Ctx, c.IP()).Result()

	if err == redis.Nil {
		_ = r2.Set(database.Ctx, c.IP(), os.Getenv("API_QUOTA"), 30*60*time.Second).Err()
	} else {
		valInt, _ := strconv.Atoi(val)

		if valInt <= 0 {
			limit, _ := r2.TTL(database.Ctx, c.IP()).Result()
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":            "Rate limit exceeded",
				"rate_limit_reset": limit / time.Nanosecond / time.Minute,
			})
		}
	}

	// check if the input if an actual URL
	if !govalidator.IsURL(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid URL"})
	}

	// check for domain error

	if !helpers.RemoveDomainError(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "URL is not Safe"})
	}

	// enforce http

	body.URL = helpers.EnforceHTTP(body.URL)

	var id string

	if body.CustomShort == "" {
		id = uuid.New().String()[:6]
	} else {
		id = body.CustomShort
	}

	dbURLStr := os.Getenv("REDIS_DB_URL")

	dbURL, err := strconv.Atoi(dbURLStr)
	if err != nil {
		dbURL = 0
	}

	r := database.CreateClient(dbURL)
	defer r.Close()

	val, _ = r.Get(database.Ctx, id).Result()

	if val != "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Your custom short is already use",
		})
	}

	if body.Expiry == 0 {
		body.Expiry = 24
	}

	err = r.Set(database.Ctx, id, body.URL, body.Expiry*3600*time.Second).Err()

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Something went wrong, please try again later",
		})
	}
	resp := response{
		URL:             body.URL,
		CustomShort:     "",
		Expiry:          body.Expiry,
		XRateRemaining:  10,
		XRateLimitReset: 30,
	}
	r2.Decr(database.Ctx, c.IP())

	val, _ = r2.Get(database.Ctx, c.IP()).Result()

	resp.XRateRemaining, _ = strconv.Atoi(val)

	ttl, _ := r2.TTL(database.Ctx, c.IP()).Result()
	resp.XRateLimitReset = ttl / time.Nanosecond / time.Minute
	resp.CustomShort = os.Getenv("DOMAIN") + "/" + id

	return c.Status(fiber.StatusOK).JSON(resp)
}
