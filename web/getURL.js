// Function to get query parameters from URL
function getQueryParams() {
	const params = {};
	const queryString = window.location.search.substring(1);
	const regex = /([^&=]+)=([^&]*)/g;
	let matches;
	while (matches = regex.exec(queryString)) {
		params[decodeURIComponent(matches[1])] = decodeURIComponent(matches[2]);
	}
	return params;
}

// Get work_id from query parameters
const queryParams = getQueryParams();
const workID = queryParams['work_id'] || "{{WORK_ID}}";
const url = new URL(`/user/work/get/${workID}`, window.location.origin);

// Make url globally available
window.generatedUrl = url;
console.log(url); // For debugging purposes
