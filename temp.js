$("a").click((e) => {
    $(".content-container").hide();    
    const containerId = $(e.target).attr("data-container");
    $(`#${containerId}`).show();
})
$(".content-container").hide();
$($("a")[0]).trigger("click");