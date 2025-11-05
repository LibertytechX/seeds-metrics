import React, { useState, useRef, useEffect } from 'react';
import { ChevronDown, X } from 'lucide-react';
import './MultiSelect.css';

const MultiSelect = ({ 
  options = [], 
  selectedValues = [], 
  onChange, 
  placeholder = "Select...",
  label = ""
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleToggle = () => {
    setIsOpen(!isOpen);
  };

  const handleOptionClick = (option) => {
    const newValues = selectedValues.includes(option)
      ? selectedValues.filter(v => v !== option)
      : [...selectedValues, option];
    onChange(newValues);
  };

  const handleClearAll = (e) => {
    e.stopPropagation();
    onChange([]);
  };

  const handleSelectAll = (e) => {
    e.stopPropagation();
    onChange(options);
  };

  const displayText = selectedValues.length === 0
    ? placeholder
    : selectedValues.length === 1
    ? selectedValues[0]
    : `${selectedValues.length} selected`;

  return (
    <div className="multi-select" ref={dropdownRef}>
      {label && <label className="multi-select-label">{label}</label>}
      <div className="multi-select-control" onClick={handleToggle}>
        <span className="multi-select-value">{displayText}</span>
        <div className="multi-select-icons">
          {selectedValues.length > 0 && (
            <button
              className="multi-select-clear"
              onClick={handleClearAll}
              title="Clear all"
            >
              <X size={14} />
            </button>
          )}
          <ChevronDown 
            size={16} 
            className={`multi-select-chevron ${isOpen ? 'open' : ''}`}
          />
        </div>
      </div>

      {isOpen && (
        <div className="multi-select-dropdown">
          <div className="multi-select-actions">
            <button 
              className="multi-select-action-btn"
              onClick={handleSelectAll}
            >
              Select All
            </button>
            <button 
              className="multi-select-action-btn"
              onClick={handleClearAll}
            >
              Clear All
            </button>
          </div>
          <div className="multi-select-options">
            {options.map((option) => (
              <div
                key={option}
                className={`multi-select-option ${selectedValues.includes(option) ? 'selected' : ''}`}
                onClick={() => handleOptionClick(option)}
              >
                <input
                  type="checkbox"
                  checked={selectedValues.includes(option)}
                  onChange={() => {}} // Handled by parent div onClick
                  onClick={(e) => e.stopPropagation()}
                />
                <span>{option}</span>
              </div>
            ))}
            {options.length === 0 && (
              <div className="multi-select-empty">No options available</div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default MultiSelect;

