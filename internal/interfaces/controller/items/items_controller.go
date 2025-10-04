package controller

import (
	"net/http"
	"strconv"

	domainErrors "Aicon-assignment/internal/domain/errors"
	"Aicon-assignment/internal/usecase"

	"github.com/labstack/echo/v4"
)

type ItemHandler struct {
	itemUsecase usecase.ItemUsecase
}

func NewItemHandler(itemUsecase usecase.ItemUsecase) *ItemHandler {
	return &ItemHandler{
		itemUsecase: itemUsecase,
	}
}

// エラーレスポンスの形式
type ErrorResponse struct {
	Error   string   `json:"error"`
	Details []string `json:"details,omitempty"`
}

func (h *ItemHandler) GetItems(c echo.Context) error {
	items, err := h.itemUsecase.GetAllItems(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "failed to retrieve items",
		})
	}

	return c.JSON(http.StatusOK, items)
}

func (h *ItemHandler) GetItem(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid item ID",
		})
	}

	item, err := h.itemUsecase.GetItemByID(c.Request().Context(), id)
	if err != nil {
		if domainErrors.IsNotFoundError(err) {
			return c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "item not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "failed to retrieve item",
		})
	}

	return c.JSON(http.StatusOK, item)
}

func (h *ItemHandler) CreateItem(c echo.Context) error {
	var input usecase.CreateItemInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid request format",
		})
	}

	// バリデーション
	if validationErrors := validateCreateItemInput(input); len(validationErrors) > 0 {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation failed",
			Details: validationErrors,
		})
	}

	item, err := h.itemUsecase.CreateItem(c.Request().Context(), input)
	if err != nil {
		if domainErrors.IsValidationError(err) {
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "validation failed",
				Details: []string{err.Error()},
			})
		}
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "failed to create item",
		})
	}

	return c.JSON(http.StatusCreated, item)
}

func (h *ItemHandler) DeleteItem(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid item ID",
		})
	}

	err = h.itemUsecase.DeleteItem(c.Request().Context(), id)
	if err != nil {
		if domainErrors.IsNotFoundError(err) {
			return c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "item not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "failed to delete item",
		})
	}

	return c.NoContent(http.StatusNoContent)
}


// 💡 新規追加: UpdateItemハンドラ
func (h *ItemHandler) UpdateItem(c echo.Context) error {
    idStr := c.Param("id")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        return c.JSON(http.StatusBadRequest, ErrorResponse{
            Error: "invalid item ID",
        })
    }

    var input usecase.UpdateItemInput
    if err := c.Bind(&input); err != nil {
        return c.JSON(http.StatusBadRequest, ErrorResponse{
            Error: "invalid request format",
        })
    }

    // バリデーション
    if validationErrors := validateUpdateItemInput(input); len(validationErrors) > 0 {
        return c.JSON(http.StatusBadRequest, ErrorResponse{
            Error:   "validation failed",
            Details: validationErrors,
        })
    }

    item, err := h.itemUsecase.UpdateItem(c.Request().Context(), id, input)
    if err != nil {
        if domainErrors.IsNotFoundError(err) {
            return c.JSON(http.StatusNotFound, ErrorResponse{
                Error: "item not found",
            })
        }
        return c.JSON(http.StatusInternalServerError, ErrorResponse{
            Error: "failed to update item",
        })
    }

    return c.JSON(http.StatusOK, item)
}



func (h *ItemHandler) GetSummary(c echo.Context) error {
	summary, err := h.itemUsecase.GetCategorySummary(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "failed to retrieve summary",
		})
	}

	return c.JSON(http.StatusOK, summary)
}

func validateCreateItemInput(input usecase.CreateItemInput) []string {
	var errs []string

	// Basic required field validation
	if input.Name == "" {
		errs = append(errs, "name is required")
	}
	if input.Category == "" {
		errs = append(errs, "category is required")
	}
	if input.Brand == "" {
		errs = append(errs, "brand is required")
	}
	if input.PurchaseDate == "" {
		errs = append(errs, "purchase_date is required")
	}
	if input.PurchasePrice < 0 {
		errs = append(errs, "purchase_price must be 0 or greater")
	}

	return errs
}

// 💡 新規追加: validateUpdateItemInput関数
func validateUpdateItemInput(input usecase.UpdateItemInput) []string {
    var errs []string
    
    // PATCHは部分更新のため、すべてのフィールドが必須ではない
    // ただし、もし提供された場合はバリデーションする
    if input.PurchasePrice != nil && *input.PurchasePrice < 0 {
        errs = append(errs, "purchase_price must be 0 or greater")
    }
    if input.Name != nil && *input.Name == "" {
        errs = append(errs, "name cannot be empty")
    }
    if input.Brand != nil && *input.Brand == "" {
        errs = append(errs, "brand cannot be empty")
    }
    
    // どのフィールドも提供されていない場合はエラーを返す
    if input.Name == nil && input.Brand == nil && input.PurchasePrice == nil {
        errs = append(errs, "at least one field (name, brand, or purchase_price) is required for update")
    }

    return errs
}