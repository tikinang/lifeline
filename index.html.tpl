<html lang="en">
<head>
    <title>My lifeline</title>
    <link title="timeline-styles" rel="stylesheet" href="https://cdn.knightlab.com/libs/timeline3/latest/css/timeline.css">
    <script src="https://cdn.knightlab.com/libs/timeline3/latest/js/timeline.js"></script>
</head>
<body>
<div id="timeline-embed"></div>
<div style="display: flex; justify-content: center; margin-top: 4rem;">
    <form method="post">
        <input style="padding: 1rem" type="submit" value="Fetch data from Google Sheets">
    </form>
</div>
<p style="text-align: center; margin-bottom: 4rem; margin-top: 2rem;">
    ⚡ Made using <a href="https://github.com/tikinang/lifeline">Lifeline</a> ⚡
</p>
<script type="text/javascript">
    const data = JSON.parse({{ .Timeline }})
    const options = {
        initial_zoom: 1,
        duration: 128,
        start_at_end: true,
        hash_bookmark: true,
    }
    window.timeline = new TL.Timeline('timeline-embed', data, options);
</script>
</body>
</html>