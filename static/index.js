var labels = document.getElementsByClassName("label");

var openIssuesPage = function(e) {
  console.log(this);
  var repoName = this.parentElement.dataset.repo;
  var labelName = this.dataset.label;
  window.open('https://github.com/'+repoName+'/labels/'+labelName);
};

for (var i = 0; i < labels.length; i++) {
  labels[i].onclick = openIssuesPage;
}

var entryCt = 50;

function limitEntries(ct) {
  var table = document.getElementById("help-wanted-table");
  var tableChildren = table.children[0].children;
  if (ct == -1) {
    ct = tableChildren.length;
  }
  for (var i = 0; i < tableChildren.length && i < ct; i++) {
    console.log(i);
    tableChildren[i].classList.remove("hidden-row");
  }
}

limitEntries(entryCt);

window.onscroll = function () {
    var totalHeight = document.documentElement.scrollHeight;
    var clientHeight = document.documentElement.clientHeight;
    var scrollTop = (document.body && document.body.scrollTop) ? document.body.scrollTop : document.documentElement.scrollTop;
    if (totalHeight == scrollTop + clientHeight) {
      entryCt += 50; 
      setTimeout(function() {
        limitEntries(entryCt);
      }, 400);
    }
};

function gotoTop() {
  window.scrollTo(0, 0);
}

function gotoBottom() {
  limitEntries(-1);
  window.scrollTo(0,document.body.scrollHeight);
}
