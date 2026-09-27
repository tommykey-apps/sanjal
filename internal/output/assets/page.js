// 埋め込んだ Mermaid の記述を、トークンの色で図にする。OS の明暗が変わったら描き直す
(function () {
  var graph = document.getElementById("graph");
  var source = document.getElementById("source").textContent;

  function cssVar(name) {
    var el = document.createElement("span");
    el.style.color = "var(" + name + ")";
    document.body.append(el);
    var v = getComputedStyle(el).color;
    el.remove();
    return v;
  }

  function showError(err) {
    graph.innerHTML = "";
    var p = document.createElement("p");
    p.className = "state state--error";
    p.textContent = "図を描けなかった。下の「Mermaid の記述」を mermaid.live に貼ると原因が分かる。" + (err && err.message ? " (" + err.message + ")" : "");
    graph.append(p);
  }

  var seq = 0;
  async function render() {
    var id = "g" + ++seq;
    try {
      await document.fonts.ready;
      window.mermaid.initialize({
        startOnLoad: false,
        securityLevel: "strict",
        theme: "base",
        fontFamily: getComputedStyle(document.documentElement).getPropertyValue("--font-family-sans").trim(),
        themeVariables: {
          primaryColor: cssVar("--color-key-50"),
          primaryTextColor: cssVar("--color-neutral-solid-gray-900"),
          primaryBorderColor: cssVar("--color-key-900"),
          lineColor: cssVar("--color-neutral-solid-gray-600"),
          textColor: cssVar("--color-neutral-solid-gray-900"),
          background: cssVar("--color-neutral-white"),
          edgeLabelBackground: cssVar("--color-neutral-white"),
        },
      });
      var out = await window.mermaid.render(id, source);
      graph.innerHTML = out.svg;
      // Mermaid は幅 100% で縮めるので、狭い画面では文字が読めなくなる。
      // 原寸に固定し、.graph の中だけ横に動かす
      var svg = graph.querySelector("svg");
      svg.style.maxWidth = "none";
      svg.style.width = svg.viewBox.baseVal.width + "px";
    } catch (err) {
      showError(err);
    }
  }

  if (!window.mermaid) {
    showError(new Error("mermaid.js を読み込めなかった"));
    return;
  }
  render();
  matchMedia("(prefers-color-scheme: dark)").addEventListener("change", render);
})();
