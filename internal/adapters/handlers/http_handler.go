package handlers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/esdrassantos06/go-shortener/internal/core/domain"
	"github.com/esdrassantos06/go-shortener/internal/core/ports"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgconn"
)

type HTTPHandler struct {
	Service ports.LinkService
	BaseURL string
}

func NewHTTPHandler(service ports.LinkService, baseURL string) *HTTPHandler {
	return &HTTPHandler{
		Service: service,
		BaseURL: baseURL,
	}
}

type CreateShortLinkRequest struct {
	TargetURL  string `json:"target_url" example:"https://example.com" binding:"required"`
	CustomSlug string `json:"custom_slug,omitempty" example:"my-custom-link"`
}

type CreateShortLinkResponse struct {
	ShortURL string      `json:"short_url" example:"http://localhost:8080/abc123"`
	Details  domain.Link `json:"details"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"Invalid input"`
}

var reservedSlugs = map[string]struct{}{
	"api":         {},
	"swagger":     {},
	"shorten":     {},
	"admin":       {},
	"health":      {},
	"metrics":     {},
	"docs":        {},
	"static":      {},
	"assets":      {},
	"favicon.ico": {},
}

func isReservedSlug(slug string) bool {
	slug = strings.ToLower(slug)
	_, exists := reservedSlugs[slug]
	return exists
}

func isDuplicateError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return strings.Contains(err.Error(), "duplicate key value violates unique constraint")
}

// CreateShortLink godoc
// @Summary      Create a shortened link
// @Description  Create a new shortened link from a URL. Requires authentication. Optionally allows defining a custom slug. Reserved slugs (api, swagger, shorten, admin, health, metrics, docs, static, assets, favicon.ico) cannot be used.
// @Tags         links
// @Accept       json
// @Produce      json
// @Param        request  body      CreateShortLinkRequest  true  "Link data"
// @Success      200      {object}  CreateShortLinkResponse  "Link created successfully"
// @Failure      400      {object}  ErrorResponse  "Validation error or reserved slug"
// @Failure      401      {object}  ErrorResponse  "Unauthorized"
// @Failure      409      {object}  ErrorResponse  "Custom slug already exists"
// @Failure      500      {object}  ErrorResponse  "Internal server error"
// @Router       /api/shorten [post]
func (h *HTTPHandler) CreateShortLink(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(ErrorResponse{
			Error: "Unauthorized: User ID not found",
		})
	}

	var req CreateShortLinkRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(400).JSON(ErrorResponse{Error: "Invalid input"})
	}

	if req.CustomSlug != "" && isReservedSlug(req.CustomSlug) {
		return c.Status(400).JSON(ErrorResponse{
			Error: fmt.Sprintf("The slug '%s' is reserved and cannot be used. Please choose a different one.", req.CustomSlug),
		})
	}

	link, err := h.Service.ShortenURL(c.Context(), req.TargetURL, req.CustomSlug, &userID)
	if err != nil {
		if isDuplicateError(err) {
			slug := req.CustomSlug
			if slug == "" {
				slug = "generated"
			}
			return c.Status(409).JSON(ErrorResponse{
				Error: fmt.Sprintf("The custom slug '%s' is already in use. Please choose a different one.", slug),
			})
		}
		return c.Status(500).JSON(ErrorResponse{Error: "An error occurred while creating the link"})
	}

	return c.JSON(CreateShortLinkResponse{
		ShortURL: fmt.Sprintf("%s/%s", h.BaseURL, link.ShortID),
		Details:  link,
	})
}

// Redirect godoc
// @Summary      Redirect to original URL
// @Description  Redirects to the original URL associated with the provided slug
// @Tags         links
// @Accept       json
// @Produce      json
// @Param        slug  path      string  true  "Shortened link slug"  example(abc123)
// @Success      301   {string}  string  "Permanent redirect"
// @Failure      404   {string}  string  "Link not found"
// @Router       /{slug} [get]
func (h *HTTPHandler) Redirect(c fiber.Ctx) error {
	slug := c.Params("slug")

	target, err := h.Service.ResolveURL(c.Context(), slug)
	if err != nil {
		return c.Status(404).SendString("Link not found")
	}

	return c.Redirect().Status(fiber.StatusMovedPermanently).To(target)
}

// ResolveSlug - Public endpoint for resolving (used by the frontend)
// @Summary      Resolve a shortened link
// @Description  Returns the target URL for a given slug. Public endpoint, no authentication required.
// @Tags         links
// @Accept       json
// @Produce      json
// @Param        slug  path      string  true  "Shortened link slug"  example(abc123)
// @Success      200   {object}  map[string]string  "Target URL"
// @Failure      404   {object}  ErrorResponse  "Link not found"
// @Router       /api/resolve/{slug} [get]
func (h *HTTPHandler) ResolveSlug(c fiber.Ctx) error {
	slug := c.Params("slug")

	target, err := h.Service.ResolveURL(c.Context(), slug)
	if err != nil {
		return c.Status(404).JSON(ErrorResponse{Error: "Link not found"})
	}

	c.Set("Cache-Control", "public, max-age=60, s-maxage=60, stale-while-revalidate=300")

	return c.JSON(fiber.Map{"target_url": target})
}
