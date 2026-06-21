/** Standard API error response. */
export interface ApiError {
  error: {
    code: string;
    message: string;
  };
}

/** Standard API success response wrapper. */
export interface ApiResponse<T> {
  data: T;
}

/** Paginated list response. */
export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  limit: number;
}

/** Pagination query parameters. */
export interface PaginationParams {
  page?: number;
  limit?: number;
}

/** Sort direction. */
export type SortDirection = "asc" | "desc";
