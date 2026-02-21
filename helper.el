;;; helper.el --- Description -*- lexical-binding: t; -*-
;;
;; Copyright (C) 2026 Duzy Chan
;;
;; Author: Duzy Chan <dc@extbit.com>
;; Maintainer: Duzy Chan <dc@extbit.com>
;; Created: February 19, 2026
;; Modified: February 19, 2026
;; Version: 0.0.1
;; Keywords: Symbol’s value as variable is void: finder-known-keywords
;; Homepage: https://github.com/duzy/helper
;; Package-Requires: ((emacs "24.3"))
;;
;; This file is not part of GNU Emacs.
;;
;;; Commentary:
;;
;;  Description
;;
;;; Code:

(require 'cl-lib)

(defun go-merge-duplicate-map-keys-1 (beg end)
  "Rewrite selected region of Go map entries.
Groups duplicates into nested maps. Leaves single entries untouched."
  (interactive "r")
  (unless (use-region-p)
    (user-error "Please select a region first"))
  
  (let* ((text (buffer-substring-no-properties beg end))
         (lines (split-string text "\n" t))
         (key-order '())
         ;; Hash table to group values: key -> (indentation . (val1 val2 ...))
         (key-data (make-hash-table :test 'equal)))
    
    ;; 1. Parse the region and group by key
    (dolist (line lines)
      (when (string-match "^\\([ \t]*\\)\\(`[^`]*`\\)[ \t]*:[ \t]*\\(`[^`]*`\\),?[ \t]*$" line)
        (let* ((indent (match-string 1 line))
               (key (match-string 2 line))
               (val (match-string 3 line))
               (existing (gethash key key-data)))
          (if existing
              ;; If key exists, prepend the new value to its list
              (puthash key (cons (car existing) (cons val (cdr existing))) key-data)
            ;; If new key, record its order and initialize its data
            (push key key-order)
            (puthash key (cons indent (list val)) key-data)))))
    
    (if (= (hash-table-count key-data) 0)
        (message "No valid `key`:`value` lines found in region.")
      
      ;; 2. Replace the region with the conditionally grouped structures
      (delete-region beg end)
      
      (dolist (key (nreverse key-order))
        (let* ((data (gethash key key-data))
               (indent (car data))
               ;; Reverse the values so they match the original top-to-bottom order
               (vals (nreverse (cdr data))))
          
          (if (= (length vals) 1)
              ;; CASE A: No duplicates. Output the original line intact (with trailing comma).
              (insert (format "%s%s: %s,\n" indent key (car vals)))
            
            ;; CASE B: Duplicates found. Output as map[string]string.
            (let ((counter 1))
              (insert (format "%s%s: map[string]string{\n" indent key))
              (dolist (v vals)
                (insert (format "%s\t`%d`: %s,\n" indent counter v))
                (cl-incf counter))
              (insert (format "%s},\n" indent))))))
      
      (message "Successfully processed %d distinct keys." (length key-order)))))

(defun go-merge-duplicate-map-keys (beg end)
  "Rewrite selected region of Go map entries containing complex/multi-line values.
Groups unique duplicates into nested `map[string]any`. 
Bypasses and preserves any surrounding text (like opening/closing braces) in the region."
  (interactive "r")
  (unless (use-region-p)
    (user-error "Please select a region first"))

  (let* ((key-order '())
         (key-data (make-hash-table :test 'equal))
         (marker-end (set-marker (make-marker) end))
         ;; Variables to hold text outside the key-value pairs
         (prefix "")
         (suffix "")
         (first-match t)
         (last-pos nil))
    
    (save-excursion
      (goto-char beg)
      
      ;; 1. Parse using forward-search
      (while (re-search-forward "^\\([ \t]*\\)\\(`[^`]*`\\)[ \t]*:[ \t]*" marker-end t)
        (let* ((match-start (match-beginning 0))
               (indent (match-string 1))
               (key (match-string 2))
               (val-start (point))
               (val-end (marker-position marker-end)))
          
          ;; Capture any surrounding text BEFORE the very first matched key
          (when first-match
            (setq prefix (buffer-substring-no-properties beg match-start))
            (setq first-match nil))
          
          ;; Look ahead to find the start of the next key
          (save-excursion
            (if (re-search-forward "^[ \t]*`[^`]*`[ \t]*:" marker-end t)
                (setq val-end (match-beginning 0))))
          
          ;; Record the end of the current value
          (setq last-pos val-end)
          
          ;; Extract and trim the value
          (let* ((raw-val (buffer-substring-no-properties val-start val-end))
                 (trimmed-val (string-trim raw-val))
                 (val (replace-regexp-in-string ",[ \t\n]*\\'" "" trimmed-val)))
            
            ;; Group into our hash table, ignoring identical values
            (let ((existing (gethash key key-data)))
              (if existing
                  (unless (member val (cdr existing))
                    (puthash key (cons (car existing) (cons val (cdr existing))) key-data))
                (push key key-order)
                (puthash key (cons indent (list val)) key-data))))
          
          ;; Advance the parser past the block we just processed
          (goto-char val-end)))
          
      ;; Capture any surrounding text AFTER the very last value
      (when last-pos
        (setq suffix (buffer-substring-no-properties last-pos marker-end))))

    ;; 2. Rewrite the grouped output
    (if (= (hash-table-count key-data) 0)
        (message "No valid `key`:`value` blocks found in region.")
      
      (delete-region beg marker-end)
      
      ;; --- RE-INSERT PREFIX ---
      (insert prefix)
      
      (dolist (key (nreverse key-order))
        (let* ((data (gethash key key-data))
               (indent (car data))
               (vals (nreverse (cdr data))))
          
          (if (= (length vals) 1)
              ;; CASE A: No unique duplicates.
              (insert (format "%s%s: %s,\n" indent key (car vals)))
            
            ;; CASE B: Multiple unique values. Output as map[string]any{}
            (let ((counter 1))
              (insert (format "%s%s: map[string]any{\n" indent key))
              (dolist (v vals)
                (insert (format "%s\t`%d`: %s,\n" indent counter v))
                (cl-incf counter))
              (insert (format "%s},\n" indent))))))
      
      ;; --- RE-INSERT SUFFIX ---
      (insert suffix)
      
      (message "Successfully processed %d distinct keys." (length key-order)))
    
    ;; Cleanup marker
    (set-marker marker-end nil)))

(provide 'helper)
;;; helper.el ends here
