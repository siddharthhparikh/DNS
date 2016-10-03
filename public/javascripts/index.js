$(document).ready(function () {

  $('#flexsearch--search').click(function (e) {
    console.log("I pressed submit button");
    $.post('/api/login', function (data, status) {
      console.log("[DATA]", data);
    });
  });
});