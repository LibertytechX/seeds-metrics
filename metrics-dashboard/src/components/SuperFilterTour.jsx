import React, { useState, useEffect, useCallback } from 'react';
import './SuperFilterTour.css';

const TOUR_EXPIRY_DAY = 20;

const TOUR_STEPS = [
  {
    target: 'year',
    title: '🎉 New: Year Filter!',
    content: 'This is a Super Filter that constrains all dashboard data to loans disbursed in the selected year.',
  },
  {
    target: 'quarter',
    title: '📅 Quarter Filter',
    content: 'Narrow down further by quarter (Q1-Q4). All other filters will operate within this date range.',
  },
];

const SuperFilterTour = ({ yearRef, quarterRef, storageKey }) => {
  const [isVisible, setIsVisible] = useState(false);
  const [currentStep, setCurrentStep] = useState(0);
  const [tooltipPosition, setTooltipPosition] = useState({ top: 0, left: 0 });

  // Check if tour should be shown
  useEffect(() => {
    const today = new Date();
    if (today.getDate() > TOUR_EXPIRY_DAY) return;

    const dismissed = localStorage.getItem(storageKey);
    if (dismissed === 'true') return;

    // Small delay to ensure refs are mounted
    const timer = setTimeout(() => setIsVisible(true), 500);
    return () => clearTimeout(timer);
  }, [storageKey]);

  // Position tooltip relative to target
  const updatePosition = useCallback(() => {
    const step = TOUR_STEPS[currentStep];
    const ref = step.target === 'year' ? yearRef : quarterRef;
    
    if (!ref?.current) return;

    const rect = ref.current.getBoundingClientRect();
    setTooltipPosition({
      top: rect.bottom + window.scrollY + 10,
      left: Math.max(10, rect.left + window.scrollX - 100),
    });
  }, [currentStep, yearRef, quarterRef]);

  useEffect(() => {
    if (isVisible) {
      updatePosition();
      window.addEventListener('resize', updatePosition);
      window.addEventListener('scroll', updatePosition);
      return () => {
        window.removeEventListener('resize', updatePosition);
        window.removeEventListener('scroll', updatePosition);
      };
    }
  }, [isVisible, updatePosition]);

  const dismissTour = () => {
    localStorage.setItem(storageKey, 'true');
    setIsVisible(false);
  };

  const nextStep = () => {
    if (currentStep < TOUR_STEPS.length - 1) {
      setCurrentStep(currentStep + 1);
    } else {
      dismissTour();
    }
  };

  const prevStep = () => {
    if (currentStep > 0) setCurrentStep(currentStep - 1);
  };

  if (!isVisible) return null;

  const step = TOUR_STEPS[currentStep];
  const targetRef = step.target === 'year' ? yearRef : quarterRef;
  const targetRect = targetRef?.current?.getBoundingClientRect();

  return (
    <>
      <div className="sft-backdrop" onClick={dismissTour} />
      {targetRect && (
        <div
          className="sft-spotlight"
          style={{
            top: targetRect.top + window.scrollY - 4,
            left: targetRect.left + window.scrollX - 4,
            width: targetRect.width + 8,
            height: targetRect.height + 8,
          }}
        />
      )}
      <div className="sft-tooltip" style={tooltipPosition}>
        <div className="sft-tooltip-arrow" />
        <h4 className="sft-title">{step.title}</h4>
        <p className="sft-content">{step.content}</p>
        <div className="sft-footer">
          <span className="sft-steps">{currentStep + 1} / {TOUR_STEPS.length}</span>
          <div className="sft-buttons">
            {currentStep > 0 && (
              <button className="sft-btn sft-btn-secondary" onClick={prevStep}>← Back</button>
            )}
            <button className="sft-btn sft-btn-skip" onClick={dismissTour}>Skip</button>
            <button className="sft-btn sft-btn-primary" onClick={nextStep}>
              {currentStep === TOUR_STEPS.length - 1 ? 'Got it!' : 'Next →'}
            </button>
          </div>
        </div>
      </div>
    </>
  );
};

export default SuperFilterTour;
