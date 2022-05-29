<html lang="en">
<head>
    <title>Tikinang's lifeline</title>
    <link title="timeline-styles" rel="stylesheet"
          href="https://cdn.knightlab.com/libs/timeline3/latest/css/timeline.css">
    <script src="https://cdn.knightlab.com/libs/timeline3/latest/js/timeline.js"></script>
</head>
<body>
<form method="post">
    <input type="submit" value="Refresh">
</form>
<div id="timeline-embed"></div>
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