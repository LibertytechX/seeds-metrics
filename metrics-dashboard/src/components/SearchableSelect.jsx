import React, { useState, useRef, useEffect } from 'react';
import { ChevronDown, X, Search } from 'lucide-react';
import './SearchableSelect.css';

const SearchableSelect = ({ 
  options = [], 
  selectedValue = '', 
  onChange, 
  placeholder = "Select...",
  label = "",
  getOptionLabel = (option) => option,
  getOptionValue = (option) => option
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const dropdownRef = useRef(null);
  const searchInputRef = useRef(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setIsOpen(false);
        setSearchTerm('');
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Focus search input when dropdown opens
  useEffect(() => {
    if (isOpen && searchInputRef.current) {
      searchInputRef.current.focus();
    }
  }, [isOpen]);

  const handleToggle = () => {
    setIsOpen(!isOpen);
    if (!isOpen) {
      setSearchTerm('');
    }
  };

  const handleOptionClick = (option) => {
    onChange(getOptionValue(option));
    setIsOpen(false);
    setSearchTerm('');
  };

  const handleClear = (e) => {
    e.stopPropagation();
    onChange('');
  };

  // Filter options based on search term
  const filteredOptions = options.filter(option => {
    const label = getOptionLabel(option).toLowerCase();
    return label.includes(searchTerm.toLowerCase());
  });

  // Get display text for selected value
  const getDisplayText = () => {
    if (!selectedValue) return placeholder;
    const selectedOption = options.find(opt => getOptionValue(opt) === selectedValue);
    return selectedOption ? getOptionLabel(selectedOption) : placeholder;
  };

  const displayText = getDisplayText();

  return (
    <div className="searchable-select" ref={dropdownRef}>
      {label && <label className="searchable-select-label">{label}</label>}
      <div className="searchable-select-control" onClick={handleToggle}>
        <span className="searchable-select-value">{displayText}</span>
        <div className="searchable-select-icons">
          {selectedValue && (
            <button
              className="searchable-select-clear"
              onClick={handleClear}
              title="Clear selection"
            >
              <X size={14} />
            </button>
          )}
          <ChevronDown 
            size={16} 
            className={`searchable-select-chevron ${isOpen ? 'open' : ''}`}
          />
        </div>
      </div>

      {isOpen && (
        <div className="searchable-select-dropdown">
          <div className="searchable-select-search">
            <Search size={16} className="search-icon" />
            <input
              ref={searchInputRef}
              type="text"
              className="searchable-select-search-input"
              placeholder="Search..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              onClick={(e) => e.stopPropagation()}
            />
          </div>
          <div className="searchable-select-options">
            {filteredOptions.map((option, index) => {
              const optionValue = getOptionValue(option);
              const optionLabel = getOptionLabel(option);
              return (
                <div
                  key={index}
                  className={`searchable-select-option ${selectedValue === optionValue ? 'selected' : ''}`}
                  onClick={() => handleOptionClick(option)}
                >
                  <span>{optionLabel}</span>
                </div>
              );
            })}
            {filteredOptions.length === 0 && (
              <div className="searchable-select-empty">
                {searchTerm ? 'No results found' : 'No options available'}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default SearchableSelect;

