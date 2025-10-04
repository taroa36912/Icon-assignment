package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"strings"

	"Aicon-assignment/internal/domain/entity"
	domainErrors "Aicon-assignment/internal/domain/errors"
)

type ItemRepository struct {
	SqlHandler
}

func (r *ItemRepository) FindAll(ctx context.Context) ([]*entity.Item, error) {
	query := `
        SELECT id, name, category, brand, purchase_price, purchase_date, created_at, updated_at
        FROM items
        ORDER BY created_at DESC
    `

	rows, err := r.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", domainErrors.ErrDatabaseError, err.Error())
	}
	defer rows.Close()

	var items []*entity.Item
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", domainErrors.ErrDatabaseError, err.Error())
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %s", domainErrors.ErrDatabaseError, err.Error())
	}

	return items, nil
}

func (r *ItemRepository) FindByID(ctx context.Context, id int64) (*entity.Item, error) {
	query := `
        SELECT id, name, category, brand, purchase_price, purchase_date, created_at, updated_at
        FROM items
        WHERE id = ?
    `

	row := r.QueryRow(ctx, query, id)

	item, err := scanItem(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domainErrors.ErrItemNotFound
		}
		return nil, fmt.Errorf("%w: %s", domainErrors.ErrDatabaseError, err.Error())
	}

	return item, nil
}

func (r *ItemRepository) Create(ctx context.Context, item *entity.Item) (*entity.Item, error) {
	query := `
        INSERT INTO items (name, category, brand, purchase_price, purchase_date)
        VALUES (?, ?, ?, ?, ?)
    `

	result, err := r.Execute(ctx, query,
		item.Name,
		item.Category,
		item.Brand,
		item.PurchasePrice,
		item.PurchaseDate,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", domainErrors.ErrDatabaseError, err.Error())
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get last insert id: %s", domainErrors.ErrDatabaseError, err.Error())
	}

	return r.FindByID(ctx, id)
}

func (r *ItemRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM items WHERE id = ?`

	result, err := r.Execute(ctx, query, id)
	if err != nil {
		return fmt.Errorf("%w: %s", domainErrors.ErrDatabaseError, err.Error())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: failed to get rows affected: %s", domainErrors.ErrDatabaseError, err.Error())
	}

	if rowsAffected == 0 {
		return domainErrors.ErrItemNotFound
	}

	return nil
}

func (r *ItemRepository) GetSummaryByCategory(ctx context.Context) (map[string]int, error) {
	query := `
        SELECT category, COUNT(*) as count
        FROM items
        GROUP BY category
    `

	rows, err := r.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", domainErrors.ErrDatabaseError, err.Error())
	}
	defer rows.Close()

	summary := make(map[string]int)
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			return nil, fmt.Errorf("%w: %s", domainErrors.ErrDatabaseError, err.Error())
		}
		summary[category] = count
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %s", domainErrors.ErrDatabaseError, err.Error())
	}

	return summary, nil
}

// 💡 新規追加: Updateメソッド (PATCH対応)
func (r *ItemRepository) Update(ctx context.Context, item *entity.Item) (*entity.Item, error) {
    // PATCHリクエストは部分更新であるため、動的にクエリを構築する
    // ここでは、更新対象フィールド（name, brand, purchase_price）がitemに設定されていると仮定する
    
    // UPDATE句とWHERE句の基本を定義
    updates := []string{}
    params := []interface{}{}

    // 更新対象フィールドのチェックとクエリの構築
    // 注意: 本来、entity.Itemはnilを許容するポインタ型フィールドを持つべきですが、
    // ここではユースケース層から渡されたitemが更新対象フィールドのみを持つと仮定して進めます。
    
    if item.Name != "" {
        updates = append(updates, "name = ?")
        params = append(params, item.Name)
    }
    if item.Brand != "" {
        updates = append(updates, "brand = ?")
        params = append(params, item.Brand)
    }
    if item.PurchasePrice >= 0 { // 0円以上の場合は更新対象とする（値が設定されたとみなす）
        updates = append(updates, "purchase_price = ?")
        params = append(params, item.PurchasePrice)
    }
    
    // updated_at は必ず更新する
    updates = append(updates, "updated_at = ?")
    params = append(params, time.Now())

    if len(updates) == 1 && updates[0] == "updated_at = ?" {
        // 更新対象フィールドがない場合は、処理を行わないかエラーとする
        // 今回はユースケース層でチェック済みのため、ここでは更新対象が少なくとも一つあると仮定する
        // 念のため、更新しない場合は元のアイテムを返す
        return r.FindByID(ctx, item.ID)
    }

    // SQLクエリの構築
    query := fmt.Sprintf("UPDATE items SET %s WHERE id = ?", strings.Join(updates, ", "))
    params = append(params, item.ID) // IDをWHERE句のパラメータとして追加

    result, err := r.Execute(ctx, query, params...)
    if err != nil {
        return nil, fmt.Errorf("%w: failed to execute update: %s", domainErrors.ErrDatabaseError, err.Error())
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return nil, fmt.Errorf("%w: failed to get rows affected: %s", domainErrors.ErrDatabaseError, err.Error())
    }

    if rowsAffected == 0 {
        return nil, domainErrors.ErrItemNotFound
    }

    // 更新後のアイテムを取得して返す
    return r.FindByID(ctx, item.ID)
}

func scanItem(scanner interface {
	Scan(dest ...interface{}) error
}) (*entity.Item, error) {
	var item entity.Item
	var purchaseDate string
	var createdAt, updatedAt time.Time

	err := scanner.Scan(
		&item.ID,
		&item.Name,
		&item.Category,
		&item.Brand,
		&item.PurchasePrice,
		&purchaseDate,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if purchaseDate != "" {
		if parsedDate, err := time.Parse("2006-01-02", purchaseDate); err == nil {
			item.PurchaseDate = parsedDate.Format("2006-01-02")
		} else {
			item.PurchaseDate = purchaseDate
		}
	}

	item.CreatedAt = createdAt
	item.UpdatedAt = updatedAt

	return &item, nil
}
