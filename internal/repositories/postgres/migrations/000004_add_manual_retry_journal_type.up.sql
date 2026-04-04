-- Add step:manual-retry to the journal_entry_type enum for manual node retry tracking.
ALTER TYPE journal_entry_type ADD VALUE 'step:manual-retry';
