// Disable a form's submit button right after it's submitted, so a slow
// double-tap (e.g. on a Like/Dislike button) can't fire the same request
// twice. This is a progressive enhancement only — every form here already
// works correctly via a plain HTML POST + redirect without any JS at all;
// this just guards against accidental double submissions.
document.addEventListener("submit", function (event) {
  var form = event.target;
  var button = form.querySelector("button[type=submit]");
  if (!button) {
    return;
  }
  // Defer to the next tick so the browser still serializes and sends the
  // form before the button becomes unusable.
  window.setTimeout(function () {
    button.disabled = true;
  }, 0);
});
