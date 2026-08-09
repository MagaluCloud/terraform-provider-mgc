package tags

// maxPageSize is the largest page the tags API serves. Asking for more is capped
// by the API, and asking for less only adds round trips.
const maxPageSize = 100

// listAllPages walks the offset pagination shared by every list endpoint of the
// tags API and returns all the items. The body carries no total count, so the end
// of the listing is a page shorter than the one asked for.
func listAllPages[T any](fetch func(limit, offset int) ([]T, error)) ([]T, error) {
	var items []T

	for offset := 0; ; offset += maxPageSize {
		page, err := fetch(maxPageSize, offset)
		if err != nil {
			return nil, err
		}

		items = append(items, page...)
		if len(page) < maxPageSize {
			return items, nil
		}
	}
}
