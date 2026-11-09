package product

// type ProductRepository struct {
// 	APIServer *server.API
// }

// func NewProductRepositor(apiServer *server.API) *ProductRepository {
// 	return &ProductRepository{
// 		APIServer: apiServer,
// 	}
// }

// func (r *ProductRepository) CreateProduct(ctx context.Context, userID string, payload *model.CreateProductPayload) (*model.Product, error) {
// 	stmt := `
// 		INSERT INTO products (title, description, price, stock_quantity, size, status, metadata, user_id)
// 		VALUES (@title, @description, @price, @stock_quantity, @size, COALESCE(@status, 'active'), @metadata, @user_id)
// 		RETURNING *
// 	`

// 	rows, err := r.APIServer.Database.Pool.Query(ctx, stmt, pgx.NamedArgs{
// 		"title":          payload.Title,
// 		"description":    payload.Description,
// 		"price":          payload.Price,
// 		"stock_quantity": payload.StockQuantity,
// 		"size":           payload.Size,
// 		"status":         payload.Status,
// 		"metadata":       payload.Metadata,
// 		"user_id":        userID,
// 	})

// 	if err != nil {
// 		return nil, fmt.Errorf("failed to execute create product query for user_id=%s", userID, err)
// 	}

// 	productItem, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Product])
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to collect created product row: %w", err)
// 	}

// 	return &productItem, nil
// }

// func (r *ProductRepository) UpdateProduct(ctx context.Context, userID string, productID uuid.UUID, payload *model.UpdateProductPayload) (*model.Product, error) {
// 	setClauses := []string{}
// 	args := pgx.NamedArgs{
// 		"id":      productID,
// 		"user_id": userID,
// 	}

// 	if payload.Title != nil {
// 		setClauses = append(setClauses, "title = @title")
// 		args["title"] = *payload.Title
// 	}
// 	if payload.Description != nil {
// 		setClauses = append(setClauses, "description = @description")
// 		args["description"] = *payload.Description
// 	}
// 	if payload.Price != nil {
// 		setClauses = append(setClauses, "price = @price")
// 		args["price"] = *payload.Price
// 	}
// 	if payload.StockQuantity != nil {
// 		setClauses = append(setClauses, "stock_quantity = @stock_quantity")
// 		args["stock_quantity"] = *payload.StockQuantity
// 	}
// 	if payload.Size != nil {
// 		setClauses = append(setClauses, "size = @size")
// 		args["size"] = *payload.Size
// 	}
// 	if payload.Status != nil {
// 		setClauses = append(setClauses, "status = @status")
// 		args["status"] = *payload.Status
// 	}
// 	if payload.Metadata != nil {
// 		setClauses = append(setClauses, "metadata = @metadata")
// 		args["metadata"] = payload.Metadata
// 	}

// 	if len(setClauses) == 0 {
// 		return nil, errs.NewBadRequestError("no fields provided for update", false, nil, nil, nil)
// 	}

// 	stmt := fmt.Sprintf(`
// 		UPDATE products
// 		SET %s
// 		WHERE id = @id AND user_id = @user_id
// 		RETURNING *
// 	`, strings.Join(setClauses, ", "))

// 	rows, err := r.APIServer.Database.Pool.Query(ctx, stmt, args)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to execute update product query for id=%s: %w", productID, err)
// 	}

// 	productItem, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Product])
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to collect updated product row: %w", err)
// 	}

// 	return &productItem, nil
// }

// func (r *ProductRepository) DeleteProduct(ctx context.Context, userID string, productID uuid.UUID) error {
// 	stmt := `
// 		DELETE FROM products
// 		WHERE id = @id AND user_id = @user_id
// 	`
// 	result, err := r.APIServer.Database.Pool.Exec(ctx, stmt, pgx.NamedArgs{
// 		"id":      productID,
// 		"user_id": userID,
// 	})
// 	if err != nil {
// 		return fmt.Errorf("failed to delete product: %w", err)
// 	}

// 	if result.RowsAffected() == 0 {
// 		code := "PRODUCT_NOT_FOUND"
// 		return errs.NewNotFoundError("todo not found", false, &code)
// 	}

// 	return nil
// }

// func (r *ProductRepository) GetProductByID(ctx context.Context, userID string, productID uuid.UUID) (*model.Product, error) {
// 	stmt := `
// 		SELECT *
// 		FROM products
// 		WHERE id = @id AND user_id = @user_id
// 	`

// 	rows, err := r.APIServer.Database.Pool.Query(ctx, stmt, pgx.NamedArgs{
// 		"id":      productID,
// 		"user_id": userID,
// 	})

// 	if err != nil {
// 		return nil, fmt.Errorf("failed to execute get product by id query for id=%s user_id=%s: %w", productID, userID, err)
// 	}

// 	productItem, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Product])
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to collect product row: %w", err)
// 	}

// 	return &productItem, nil
// }

// func (r *ProductRepository) GetProducts(ctx context.Context, userID string, query *model.GetProductByQuery) ([]model.Product, error) {
// 	// 1. Base condition
// 	whereClauses := []string{"user_id = @user_id"}
// 	args := pgx.NamedArgs{
// 		"user_id": userID,
// 	}

// 	// 2. Apply Standard Filters
// 	if query.Status != nil {
// 		whereClauses = append(whereClauses, "status = @status")
// 		args["status"] = *query.Status
// 	}
// 	if query.MinPrice != nil {
// 		whereClauses = append(whereClauses, "price >= @min_price")
// 		args["min_price"] = *query.MinPrice
// 	}
// 	if query.MaxPrice != nil {
// 		whereClauses = append(whereClauses, "price <= @max_price")
// 		args["max_price"] = *query.MaxPrice
// 	}
// 	if query.Size != nil {
// 		whereClauses = append(whereClauses, "size = @size")
// 		args["size"] = *query.Size
// 	}

// 	// 3. Apply Extensions
// 	if query.InStockOnly != nil && *query.InStockOnly {
// 		// No NamedArg needed for a static condition
// 		whereClauses = append(whereClauses, "stock_quantity > 0")
// 	}

// 	if query.Search != nil {
// 		// ILIKE handles case-insensitive substring searching
// 		whereClauses = append(whereClauses, "(title ILIKE @search OR description ILIKE @search)")
// 		// Add wildcard % around the search term
// 		args["search"] = "%" + *query.Search + "%"
// 	}

// 	// 4. Pagination (Relying on defaults set by your DTO's Validate method)
// 	limit := *query.Limit
// 	offset := (*query.Page - 1) * limit
// 	args["limit"] = limit
// 	args["offset"] = offset

// 	// 5. Build Final Query
// 	stmt := fmt.Sprintf(`
// 		SELECT *
// 		FROM products
// 		WHERE %s
// 		ORDER BY created_at DESC
// 		LIMIT @limit OFFSET @offset
// 	`, strings.Join(whereClauses, " AND "))

// 	// 6. Execute and Collect
// 	rows, err := r.APIServer.Database.Pool.Query(ctx, stmt, args)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to execute get products query: %w", err)
// 	}

// 	productsList, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Product])
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to collect product rows: %w", err)
// 	}

// 	// 7. Calculate Total Pages and Return
// 	totalPages := 0
// 	if total > 0 {
// 		totalPages = (total + limit - 1) / limit
// 	}

// 	return &model.PaginatedResponse[model.Product]{
// 		Data:       productsList,
// 		Page:       *query.Page,
// 		Limit:      limit,
// 		Total:      total,
// 		TotalPages: totalPages,
// 	}, nil

// }
