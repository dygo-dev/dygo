package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// MetadataPage is one persisted app-owned Page definition exposed to Studio.
type MetadataPage struct {
	ID          int64           `json:"-"`
	Name        string          `json:"name"`
	Key         string          `json:"key"`
	Label       string          `json:"label"`
	Description string          `json:"description"`
	Icon        string          `json:"icon,omitempty"`
	Path        string          `json:"path"`
	Renderer    string          `json:"renderer"`
	Options     json.RawMessage `json:"options,omitempty"`
	App         MetadataAppRef  `json:"app"`
}

// ListPages returns active Pages ordered by app and Page key.
func (r MetadataReader) ListPages(ctx context.Context) ([]MetadataPage, error) {
	if err := r.requireQueryer(); err != nil {
		return nil, err
	}
	rows, err := r.queryer.Query(ctx, `
SELECT p.id, p.name, p.key, p.label, COALESCE(p.description, ''), COALESCE(p.icon, ''), p.path, p.renderer, p.options, a.name, a.label
FROM "page" p
JOIN "app" a ON a.id = p.app_id
WHERE COALESCE(p.retired, false) = false
ORDER BY a.name, p.key`)
	if err != nil {
		return nil, fmt.Errorf("query metadata pages: %w", err)
	}
	defer rows.Close()

	pages := []MetadataPage{}
	for rows.Next() {
		var page MetadataPage
		var options []byte
		if err := rows.Scan(&page.ID, &page.Name, &page.Key, &page.Label, &page.Description, &page.Icon, &page.Path, &page.Renderer, &options, &page.App.Name, &page.App.Label); err != nil {
			return nil, fmt.Errorf("scan metadata page: %w", err)
		}
		page.Options = rawJSONOrNil(options)
		pages = append(pages, page)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read metadata pages: %w", err)
	}
	return pages, nil
}

// GetPage returns one active Page by app-scoped identity.
func (r MetadataReader) GetPage(ctx context.Context, appName string, pageKey string) (MetadataPage, error) {
	if err := r.requireQueryer(); err != nil {
		return MetadataPage{}, err
	}
	var page MetadataPage
	var options []byte
	err := r.queryer.QueryRow(ctx, `
SELECT p.id, p.name, p.key, p.label, COALESCE(p.description, ''), COALESCE(p.icon, ''), p.path, p.renderer, p.options, a.name, a.label
FROM "page" p
JOIN "app" a ON a.id = p.app_id
WHERE a.name = $1 AND p.key = $2 AND COALESCE(p.retired, false) = false`, appName, pageKey).Scan(
		&page.ID, &page.Name, &page.Key, &page.Label, &page.Description, &page.Icon, &page.Path, &page.Renderer, &options, &page.App.Name, &page.App.Label,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return MetadataPage{}, MetadataNotFoundError{Kind: "page", Name: appName + "/" + pageKey}
	}
	if err != nil {
		return MetadataPage{}, fmt.Errorf("query metadata page %s/%s: %w", appName, pageKey, err)
	}
	page.Options = rawJSONOrNil(options)
	return page, nil
}
