import React from 'react';
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from 'lucide-react';
import './Pagination.css';

/**
 * Reusable Pagination Component
 * 
 * @param {Object} props
 * @param {number} props.currentPage - Current page number (1-based)
 * @param {number} props.totalPages - Total number of pages
 * @param {number} props.totalRecords - Total number of records
 * @param {number} props.pageSize - Number of records per page
 * @param {Function} props.onPageChange - Callback when page changes
 * @param {Function} props.onPageSizeChange - Callback when page size changes
 * @param {Array<number>} props.pageSizeOptions - Available page size options
 * @param {boolean} props.showTopPagination - Show pagination at top (default: false)
 * @param {boolean} props.loading - Loading state (default: false)
 */
const Pagination = ({
  currentPage = 1,
  totalPages = 1,
  totalRecords = 0,
  pageSize = 50,
  onPageChange,
  onPageSizeChange,
  pageSizeOptions = [10, 25, 50, 100, 200],
  showTopPagination = false,
  loading = false,
  position = 'bottom' // 'top' or 'bottom'
}) => {
  const startRecord = totalRecords === 0 ? 0 : (currentPage - 1) * pageSize + 1;
  const endRecord = Math.min(currentPage * pageSize, totalRecords);

  const handlePageChange = (newPage) => {
    if (newPage >= 1 && newPage <= totalPages && newPage !== currentPage && !loading) {
      onPageChange(newPage);
    }
  };

  const handlePageSizeChange = (e) => {
    const newSize = parseInt(e.target.value);
    if (onPageSizeChange && !loading) {
      onPageSizeChange(newSize);
    }
  };

  // Generate page numbers to display
  const getPageNumbers = () => {
    const pages = [];
    const maxPagesToShow = 7;

    if (totalPages <= maxPagesToShow) {
      // Show all pages if total is small
      for (let i = 1; i <= totalPages; i++) {
        pages.push(i);
      }
    } else {
      // Show first page, last page, current page, and surrounding pages
      if (currentPage <= 4) {
        // Near the beginning
        for (let i = 1; i <= 5; i++) {
          pages.push(i);
        }
        pages.push('...');
        pages.push(totalPages);
      } else if (currentPage >= totalPages - 3) {
        // Near the end
        pages.push(1);
        pages.push('...');
        for (let i = totalPages - 4; i <= totalPages; i++) {
          pages.push(i);
        }
      } else {
        // In the middle
        pages.push(1);
        pages.push('...');
        for (let i = currentPage - 1; i <= currentPage + 1; i++) {
          pages.push(i);
        }
        pages.push('...');
        pages.push(totalPages);
      }
    }

    return pages;
  };

  const pageNumbers = getPageNumbers();

  return (
    <div className={`pagination-container ${position}`}>
      <div className="pagination-info">
        <div className="rows-per-page">
          <label htmlFor={`page-size-${position}`}>Rows per page:</label>
          <select
            id={`page-size-${position}`}
            value={pageSize}
            onChange={handlePageSizeChange}
            disabled={loading}
            className="page-size-select"
          >
            {pageSizeOptions.map(option => (
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </select>
        </div>
        <div className="record-info">
          {totalRecords > 0 ? (
            <>
              Showing <strong>{startRecord}</strong> to <strong>{endRecord}</strong> of{' '}
              <strong>{totalRecords.toLocaleString()}</strong> records
            </>
          ) : (
            'No records found'
          )}
        </div>
      </div>

      <div className="pagination-controls">
        <button
          className="pagination-btn first"
          onClick={() => handlePageChange(1)}
          disabled={currentPage === 1 || loading}
          title="First Page"
          aria-label="First Page"
        >
          <ChevronsLeft size={16} />
        </button>
        <button
          className="pagination-btn prev"
          onClick={() => handlePageChange(currentPage - 1)}
          disabled={currentPage === 1 || loading}
          title="Previous Page"
          aria-label="Previous Page"
        >
          <ChevronLeft size={16} />
          <span className="btn-text">Previous</span>
        </button>

        <div className="page-numbers">
          {pageNumbers.map((page, index) => (
            page === '...' ? (
              <span key={`ellipsis-${index}`} className="page-ellipsis">
                ...
              </span>
            ) : (
              <button
                key={page}
                className={`page-number ${currentPage === page ? 'active' : ''}`}
                onClick={() => handlePageChange(page)}
                disabled={loading}
                aria-label={`Page ${page}`}
                aria-current={currentPage === page ? 'page' : undefined}
              >
                {page}
              </button>
            )
          ))}
        </div>

        <button
          className="pagination-btn next"
          onClick={() => handlePageChange(currentPage + 1)}
          disabled={currentPage === totalPages || loading}
          title="Next Page"
          aria-label="Next Page"
        >
          <span className="btn-text">Next</span>
          <ChevronRight size={16} />
        </button>
        <button
          className="pagination-btn last"
          onClick={() => handlePageChange(totalPages)}
          disabled={currentPage === totalPages || loading}
          title="Last Page"
          aria-label="Last Page"
        >
          <ChevronsRight size={16} />
        </button>
      </div>

      <div className="page-info-compact">
        Page <strong>{currentPage}</strong> of <strong>{totalPages}</strong>
      </div>
    </div>
  );
};

export default Pagination;

