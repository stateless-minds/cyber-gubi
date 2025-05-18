  // Expose a global function to call setTimeout with a Go callback
  window.goAppSetTimeout = function(cb, delay) {
    return setTimeout(cb, delay);
  };

  window.goAppClearTimeout = function(id) {
    clearTimeout(id);
  };
