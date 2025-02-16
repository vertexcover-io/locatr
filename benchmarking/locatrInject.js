/**
* Gets the index information for an element among its siblings
* @param {Element} element - The element to get index for
* @returns {number} The element's index
*/
function getElementIndex(element) {

const tag = element.tagName.toLowerCase();
let index = 1;
let prevSibling = element.previousElementSibling;

while (prevSibling) {
  if (prevSibling.tagName.toLowerCase() === tag) {
    index++;
  }
  prevSibling = prevSibling.previousElementSibling;
}

return index;
}

/**
* Gets absolute XPath for an element with caching
* @param {Element} element - The element to get XPath for
* @returns {string} The absolute XPath
*/
function getAbsXpath(element) {

const parts = [];
let current = element;

while (current && current.nodeType === Node.ELEMENT_NODE) {
  const tag = current.tagName.toLowerCase();
  const index = getElementIndex(current);
  parts.unshift(index > 1 ? `${tag}[${index}]` : tag);
  current = current.parentElement;
}

const xpath = '/' + parts.join('/');

return xpath;
}


const generateUniqueLocatorsFromCoords = (x, y) => {
  const element = document.elementFromPoint(x, y);
  const absXpath = getAbsXpath(element);
  return [absXpath];
}
