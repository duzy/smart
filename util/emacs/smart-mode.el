;;; smart-mode.el --- smart file editing commands for Emacs -*- lexical-binding:t -*-

;; Copyright (C) 2016 Duzy Chan <code@duzy.info>, http://duzy.info

;; Author: Duzy Chan <code@duzy.info>
;; Maintainer: code@duzy.info
;; Keywords: unix, tools
(require 'make-mode)

(defgroup smart nil
  "Smart editing commands for Emacs."
  :link '(custom-group-link :tag "Font Lock Faces group" font-lock-faces)
  :group 'tools
  :prefix "smart-")

(defface smart-module-name-face
  '((t :inherit font-lock-variable-name-face)) ;; :background  "LightBlue1"
  "Face to use for additionally highlighting rule targets in Font-Lock mode."
  :group 'smart)

;; http://raebear.net/comp/emacscolors.html
(defface smart-receipt-indent-face
  '((((class color) (background light)) :background "gray88" :italic t)
    (((class color) (background dark)) :background "LightDim" :italic t)
    (t :inherit font-lock-constant-face))
  "Face to use for additionally highlighting rule targets in Font-Lock mode."
  :group 'smart)

(defface smart-receipt-face '((t :background "gray96"))
  "Face to use for additionally highlighting rule targets in Font-Lock mode."
  :group 'smart)

(defvar smart-receipt-indent-face 'smart-receipt-indent-face)
(defvar smart-receipt-face 'smart-receipt-face)

(defconst smart-var-use-regex
  "[^$]\\$[({]\\([-a-zA-Z0-9_.]+\\|[@%<?^+*][FD]?\\)"
  "Regex used to find $(macro) uses in a makefile.")

(defconst smart-statements
  `("project" "module" "template" "files" "extensions" "dialect" "instance" "import" "use" "include" "eval" "export"
    ,@(cdr makefile-statements))
  "List of keywords understood by smart.")

(defconst smart-font-lock-keywords
  (makefile-make-font-lock-keywords
   smart-var-use-regex
   smart-statements
   t
   "^\\(?: [ \t]*\\)?if\\(n\\)\\(?:def\\|eq\\)\\>"

   '("[^$]\\(\\$[({][@%*][DF][})]\\)"
     1 'makefile-targets append)

   ;; $(function ...) ${function ...}
   '("[^$]\\$[({]\\([-a-zA-Z0-9_.]+\\s \\)"
     1 font-lock-function-name-face prepend)

   ;; $(shell ...) ${shell ...}
   '("[^$]\\$\\([({]\\)shell[ \t]+"
     makefile-match-function-end nil nil
     (1 'makefile-shell prepend t))

   ;; $(template ...) $(module ...)
   '("[^$]\\$[({]\\(template\\|module\\|use\\)[ \t]\\([^,)}]+\\)"
     (1 font-lock-builtin-face prepend)
     (2 font-lock-string-face prepend))

   ;; $(commit ...)
   '("[^$]\\$[({]\\(commit\\|post\\)[ \t)}]"
     1 font-lock-builtin-face prepend)

   ;; recipts
   '("^\\(\t\\)\\(.+\\)"
     (1 smart-receipt-indent-face prepend)
     (2 smart-receipt-face prepend)
     )
   ))

(define-derived-mode smart-mode makefile-gmake-mode "smart"
  "Major mode for editing .smart files."
  (setq font-lock-defaults
	`(smart-font-lock-keywords ,@(cdr font-lock-defaults)))

  ;; Real TABs are important
  (setq indent-tabs-mode t))


(progn (add-to-list 'auto-mode-alist '("\\.smart" . smart-mode))
       (add-to-list 'auto-mode-alist '("\\.sm" . smart-mode))
       (message "smart-mode"))

(provide 'smart-mode)
