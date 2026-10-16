package tags

const maxPageSize = 100

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
