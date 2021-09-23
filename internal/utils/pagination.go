package utils

type Paginate struct {
	Start_page     int `json:"start_page"`
	Items_on_page  int `json:"items_on_page,omitempty"`
	Items_per_page int `json:"items_per_page,omitempty"`
	Total_result   int `json:"total_result,omitempty"`
}

func PaginateEndPoint(sizeResp, count, startPage int) (paginate Paginate, indexS, indexE int) {
	total_result := sizeResp
	items_per_page := count
	items_on_page := 0
	start_page := startPage

	start_index := -1
	end_index := sizeResp

	if count >= 0 && startPage >= 0 {
		first_item := startPage * count
		last_item := first_item + count
		if first_item < sizeResp {
			if last_item < sizeResp {
				start_index = first_item
				end_index = last_item
			} else {
				start_index = first_item
			}
		}
	}
	if start_index >= 0 {
		items_on_page = end_index - start_index
	}
	return NewPaginate(start_page, items_on_page, items_per_page, total_result), start_index, end_index
}

func NewPaginate(startPage, itemsOnPage, itemsPerPage, totalResult int) Paginate {
	return Paginate{
		Start_page:     startPage,
		Items_on_page:  itemsOnPage,
		Items_per_page: itemsPerPage,
		Total_result:   totalResult,
	}
}
