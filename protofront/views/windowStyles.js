const windowStyles = `
  .desktop-window {
    position: absolute;
    background: rgba(32, 32, 32, 0.95);
    backdrop-filter: blur(20px);
    border: 1px solid rgba(255, 255, 255, 0.2);
    border-radius: 12px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
    display: flex;
    flex-direction: column;
    overflow: hidden;
    transition: all 0.2s ease;
    min-width: 300px;
    min-height: 200px;
  }

  .desktop-window.maximized {
    border-radius: 0;
  }

  .desktop-window.minimized {
    transform: scale(0.9);
    opacity: 0;
  }

  .desktop-window.closing {
    transform: scale(0.8);
    opacity: 0;
  }

  .window-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;
    background: rgba(255, 255, 255, 0.1);
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
    cursor: move;
    user-select: none;
  }

  .window-title {
    display: flex;
    align-items: center;
    gap: 8px;
    color: white;
    font-size: 14px;
    font-weight: 500;
  }

  .window-controls {
    display: flex;
    gap: 8px;
  }

  .window-btn {
    background: none;
    border: none;
    color: white;
    cursor: pointer;
    width: 28px;
    height: 28px;
    border-radius: 6px;
    font-size: 14px;
    transition: all 0.2s;
  }

  .window-btn:hover {
    background: rgba(255, 255, 255, 0.2);
  }

  .window-close:hover {
    background: #e81123;
  }

  .window-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    padding: 16px;
  }

  .window-toolbar {
    display: flex;
    gap: 8px;
    margin-bottom: 16px;
    padding-bottom: 12px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  }

  .btn-window {
    background: rgba(255, 255, 255, 0.1);
    border: 1px solid rgba(255, 255, 255, 0.2);
    color: white;
    padding: 6px 12px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 13px;
    transition: all 0.2s;
  }

  .btn-window:hover:not(:disabled) {
    background: rgba(255, 255, 255, 0.2);
    transform: translateY(-1px);
  }

  .btn-window:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .table-container {
    flex: 1;
    overflow: auto;
    background: rgba(0, 0, 0, 0.3);
    border-radius: 8px;
  }

  .data-table {
    width: 100%;
    border-collapse: collapse;
    color: white;
  }

  .data-table thead {
    background: rgba(50, 50, 50, 0.9);
    position: sticky;
    top: 0;
  }

  .data-table th,
  .data-table td {
    padding: 10px 12px;
    text-align: left;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  }

  .data-table tbody tr:hover {
    background: rgba(255, 255, 255, 0.05);
    cursor: pointer;
  }

  .data-table tbody tr.selected {
    background: rgba(255, 255, 255, 0.1);
  }

  .window-modal {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10000;
  }

  .window-modal-content {
    background: rgba(32, 32, 32, 0.95);
    backdrop-filter: blur(20px);
    border-radius: 12px;
    min-width: 400px;
    max-width: 500px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  }

  .window-modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  }

  .window-modal-header h3 {
    margin: 0;
    color: white;
    font-size: 16px;
  }

  .modal-close {
    background: none;
    border: none;
    color: white;
    font-size: 24px;
    cursor: pointer;
    padding: 0;
    width: 30px;
    height: 30px;
    border-radius: 6px;
  }

  .modal-close:hover {
    background: rgba(255, 255, 255, 0.2);
  }

  .window-modal-body {
    padding: 20px;
  }

  .form-group {
    margin-bottom: 16px;
  }

  .form-group label {
    display: block;
    color: white;
    margin-bottom: 6px;
    font-size: 13px;
    font-weight: 500;
  }

  .form-group input,
  .form-group select,
  .form-group textarea {
    width: 100%;
    padding: 8px 12px;
    background: rgba(255, 255, 255, 0.1);
    border: 1px solid rgba(255, 255, 255, 0.2);
    border-radius: 6px;
    color: white;
    font-size: 14px;
    box-sizing: border-box;
  }

  .form-group input:focus,
  .form-group select:focus,
  .form-group textarea:focus {
    outline: none;
    border-color: #0078d4;
  }

  .window-modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding: 16px 20px;
    border-top: 1px solid rgba(255, 255, 255, 0.1);
  }

  .window-modal-footer button {
    padding: 8px 16px;
    border-radius: 6px;
    border: none;
    cursor: pointer;
    font-size: 13px;
    transition: all 0.2s;
  }

  .btn-cancel {
    background: rgba(255, 255, 255, 0.1);
    color: white;
  }

  .btn-cancel:hover {
    background: rgba(255, 255, 255, 0.2);
  }

  .btn-save {
    background: #0078d4;
    color: white;
  }

  .btn-save:hover {
    background: #005a9e;
  }

  .window-resize-handle {
    position: absolute;
    bottom: 0;
    right: 0;
    width: 15px;
    height: 15px;
    cursor: se-resize;
    background: linear-gradient(135deg, transparent 50%, rgba(255, 255, 255, 0.3) 50%);
    border-bottom-right-radius: 12px;
  }

  .status-cell {
    width: 8px;
    height: 8px;
    color: transparent; /* Текст станет невидимым */
    overflow: hidden;    /* На всякий случай */
    user-select: none;   /* Чтобы нельзя было случайно выделить цифру */
    filter: invert(1)
  }

  /* Если значение 1 — иконка листика (используем Unicode) */
  .status-cell[data-value="1"] {
    background-image: url('png/file.png');
    background-repeat: no-repeat;
    background-position: center;
    background-size: contain;
  }
  /* Если значение 0 — иконка с крестиком */
  .status-cell[data-value="0"] {
    background-image: url('png/delete-document.png');
    background-repeat: no-repeat;
    background-position: center;
    background-size: contain;  }

    .status-cell[data-value="2"] {
    background-image: url('png/assept-document.png');
    background-repeat: no-repeat;
    background-position: center;
    background-size: contain;  }

    /* Современные стили для select */
    .custom-select {
        width: 100%;
        padding: 10px;
        border: 2px solid #e0e0e0;
        border-radius: 8px;
        font-size: 16px;
        background-color: white;
        cursor: pointer;
        transition: all 0.3s ease;
    }

    .custom-select:hover {
        border-color: #007bff;
    }

    .custom-select:focus {
        outline: none;
        border-color: #007bff;
        box-shadow: 0 0 0 3px rgba(0,123,255,0.1);
    }

    .select-search {
        width: 100%;
        padding: 8px;
        margin-bottom: 10px;
        border: 1px solid #ddd;
        border-radius: 4px;
        font-size: 14px;
    }

    .enhanced-select {
        margin: 20px 0;
    }

    /* Для множественного выбора */
    .custom-select[multiple] {
        min-height: 150px;
    }

    .custom-select option {
        padding: 8px;
        margin: 2px 0;
    }

    .custom-select option:checked {
        background: #007bff linear-gradient(0deg, #007bff 0%, #007bff 100%);
        color: white;
    }
    .form-input {
      width: 100%;
      padding: 8px 12px;
      background: rgba(255, 255, 255, 0.1);
      border: 1px solid rgba(255, 255, 255, 0.2);
      border-radius: 6px;
      color: white;
      font-size: 14px;
      box-sizing: border-box;
    }

    .form-input:focus {
      outline: none;
      border-color: #0078d4;
    }

    .enhanced-select-wrapper {
      margin-top: 5px;
    }
`;

// Add styles to document
if (!document.querySelector("#window-styles")) {
  const styleSheet = document.createElement("style");
  styleSheet.id = "window-styles";
  styleSheet.textContent = windowStyles;
  document.head.appendChild(styleSheet);
}
